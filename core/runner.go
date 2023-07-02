package core

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

type Runner struct {
	config   Config
	ticker   *time.Ticker
	requests []*Request

	// concurrency sync
	ctx    context.Context
	Cancel context.CancelFunc
	wg     *sync.WaitGroup

	// request stats
	reqCount     uint64
	resCount     uint64
	reqCountChan chan struct{}
	resCountChan chan struct{}
	ReqPS        chan uint64
	ResPS        chan uint64

	// error stats
	errorCount   uint64
	errIn        chan interface{}
	ErrOut       chan interface{}
	ErrCountChan chan uint64

	// response time stats
	resTimesIn  chan uint64
	ResTimesOut chan uint64
	ResStats    chan ResponseTimeStats

	// shutdown signal
	Done chan struct{}
}

type ResponseTimeStats struct {
	AverageTime uint64
	MaxTime     uint64
	MinTime     uint64
}

func NewRunner(config Config, ticker *time.Ticker) *Runner {

	// TODO: Check for closing of all channels

	return &Runner{
		config:       config,
		ticker:       ticker,
		wg:           &sync.WaitGroup{},
		reqCountChan: make(chan struct{}, config.NumClients),
		resCountChan: make(chan struct{}, config.NumClients),
		errIn:        make(chan interface{}, config.NumClients),
		errOut:       make(chan interface{}, config.NumClients),
		errCountChan: make(chan uint64),
		resTimesIn:   make(chan uint64, config.NumClients),
		resTimesOut:  make(chan uint64, config.NumClients),
		resStats:     make(chan ResponseTimeStats, config.NumClients),
		reqPS:        make(chan uint64),
		resPS:        make(chan uint64),
		done:         make(chan struct{}),
	}
}

// TODO: Error handling logic should be extracted and left to the main function

func (r *Runner) getRequestSpec() {
	ext := filepath.Ext(r.config.ReqSpecPath)
	if ext != ".json" {
		log.Fatal("Invalid file format. Expected a JSON file.")
	}

	reqSpec, err := os.Open(r.config.ReqSpecPath)
	if err != nil {
		log.Fatalf("Could not open file: %v", err)
	}

	defer reqSpec.Close()

	bytes, err := io.ReadAll(reqSpec)
	if err != nil {
		log.Fatalf("Error reading file: %v", err)
	}

	err = json.Unmarshal(bytes, &r.requests)
	if err != nil {
		log.Fatalf("Error parsing json file: %v", err)
	}
}

func (r *Runner) validateRequests() {
	validRequests := make([]*Request, 0)

	for _, req := range r.requests {
		switch req.Verb {
		case "GET", "POST", "PUT", "DELETE":
			bodyBytes, err := json.Marshal(req.Body)
			if err != nil {
				log.Printf("Error: could not parse request body for verb: %s\turl: %s\n", req.Verb, req.URL)
				continue
			}
			req.BodyBytes = bodyBytes
			validRequests = append(validRequests, req)
		default:
			log.Printf("Error: verb: %s not allowed. Only GET, POST, PUT, DELETE are allowed\n", req.Verb)
			continue
		}
	}

	log.Printf("INFO:  total requests: %d\t valid requests: %d\n", len(r.requests), len(validRequests))

	r.requests = validRequests
}

func (r *Runner) initCounters() {
	r.wg.Add(1)

	// reqs counter
	go func(ctx context.Context) {
		defer r.wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-r.reqCountChan:
				if !ok {
					return
				}
				atomic.AddUint64(&r.reqCount, 1)
			}
		}
	}(r.ctx)

	r.wg.Add(1)

	// res counter
	go func(ctx context.Context) {
		defer r.wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-r.resCountChan:
				if !ok {
					return
				}
				atomic.AddUint64(&r.resCount, 1)
			}
		}
	}(r.ctx)

	r.wg.Add(1)

	// error counter
	go func(ctx context.Context) {
		defer r.wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			case err, ok := <-r.errIn:
				if !ok {
					return
				}
				atomic.AddUint64(&r.errorCount, 1)
				r.errCountChan <- r.errorCount
				r.errOut <- err
			}
		}
	}(r.ctx)
}

func (r *Runner) getRPS(ticker *time.Ticker) {
	r.wg.Add(1)

	go func(ctx context.Context) {
		defer r.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-ticker.C:
				if !ok {
					return
				}
				reqC := atomic.SwapUint64(&r.reqCount, 0)
				r.reqPS <- reqC
				resC := atomic.SwapUint64(&r.resCount, 0)
				r.resPS <- resC
			}
		}
	}(r.ctx)
}

func (r *Runner) getResponseTimesStats() {
	r.wg.Add(1)

	go func(ctx context.Context) {
		defer r.wg.Done()

		var numRes uint64
		var resTimesSum uint64
		var maxResTime uint64
		var minResTime uint64 = math.MaxUint64
		var avgResTime uint64
		stats := ResponseTimeStats{
			AverageTime: avgResTime,
			MaxTime:     maxResTime,
			MinTime:     minResTime,
		}

		for {
			select {
			case <-ctx.Done():
				return
			case resTime, ok := <-r.resTimesIn:
				if !ok {
					return
				}

				r.resTimesOut <- resTime

				atomic.AddUint64(&numRes, 1)
				atomic.AddUint64(&resTimesSum, resTime)
				avg := resTimesSum / numRes
				atomic.SwapUint64(&avgResTime, avg)

				if resTime > maxResTime {
					atomic.SwapUint64(&maxResTime, resTime)
				}

				if resTime < minResTime {
					atomic.SwapUint64(&minResTime, resTime)
				}

				newStats := ResponseTimeStats{
					AverageTime: avgResTime,
					MaxTime:     maxResTime,
					MinTime:     minResTime,
				}

				if !reflect.DeepEqual(stats, newStats) {
					r.resStats <- newStats
					stats = newStats
				}
			}
		}
	}(r.ctx)
}

func (r *Runner) LoadTest() (chan struct{}, context.CancelFunc, chan uint64, chan uint64, chan uint64, chan ResponseTimeStats, chan interface{}, chan uint64) {
	r.getRequestSpec()

	r.validateRequests()

	duration := r.config.Duration
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	r.ctx = ctx

	r.getRPS(r.ticker)

	r.initCounters()

	r.getResponseTimesStats()

	for i := 0; i < r.config.NumClients; i++ {
		client := newClient(r.requests, r.ctx, r.wg, r.reqCountChan, r.resCountChan, r.resTimesIn, r.errIn)
		client.start()
	}

	// wait for all the goroutines to exit
	go func() {
		r.wg.Wait()

		// close data channels
		close(r.reqCountChan)
		close(r.resCountChan)
		close(r.errIn)
		close(r.errOut)
		close(r.errCountChan)
		close(r.resTimesIn)
		close(r.resTimesOut)
		close(r.resStats)
		close(r.reqPS)
		close(r.resPS)

		// finally close main done channel
		close(r.done)
	}()

	return r.done, cancel, r.reqPS, r.resPS, r.resTimesOut, r.resStats, r.errOut, r.errCountChan
}
