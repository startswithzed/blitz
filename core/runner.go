package core

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

type Runner struct {
	config       Config
	ticker       *time.Ticker
	requests     []*Request
	ctx          context.Context
	wg           *sync.WaitGroup
	reqCount     uint64
	resCount     uint64
	errorCount   uint64
	reqCountChan chan struct{}
	resCountChan chan struct{}
	errorStream  chan ErrorLog
	reqPS        chan uint64
	resPS        chan uint64
	done         chan struct{}
}

func NewRunner(config Config, ticker *time.Ticker) *Runner {

	// TODO: Check for closing of all channels

	return &Runner{
		config:       config,
		ticker:       ticker,
		wg:           &sync.WaitGroup{},
		reqCountChan: make(chan struct{}, config.NumClients),
		resCountChan: make(chan struct{}, config.NumClients),
		errorStream:  make(chan ErrorLog),
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

func (r *Runner) countErrors() {
	r.wg.Add(1)

	go func(ctx context.Context) {
		defer r.wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-r.errorStream:
				if !ok {
					return
				}
				atomic.AddUint64(&r.errorCount, 1)
			}
		}
	}(r.ctx)
}

func (r *Runner) LoadTest() (chan struct{}, chan uint64, chan uint64) {
	r.getRequestSpec()

	r.validateRequests()

	duration := r.config.Duration
	ctx, _ := context.WithTimeout(context.Background(), duration)
	r.ctx = ctx

	r.getRPS(r.ticker)

	r.initCounters()

	for i := 0; i < r.config.NumClients; i++ {
		client := newClient(r.requests, r.ctx, r.wg, r.reqCountChan, r.resCountChan)
		client.start()
	}

	// wait for all the goroutines to exit
	go func() {
		r.wg.Wait()
		close(r.done)
	}()

	return r.done, r.reqPS, r.resPS
}
