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
	requests     []*Request
	ctx          context.Context
	wg           *sync.WaitGroup
	reqCount     uint64
	resCount     uint64
	errorCount   uint64
	reqCountChan chan struct{}
	resCountChan chan struct{}
	errorStream  chan ErrorLog
}

func NewRunner(config Config) *Runner {

	return &Runner{
		config:       config,
		wg:           &sync.WaitGroup{},
		reqCountChan: make(chan struct{}, config.NumClients),
		resCountChan: make(chan struct{}, config.NumClients),
		errorStream:  make(chan ErrorLog),
	}
}

// TODO: Error handling logic should be extracted and left to the main function
// TODO: Use wait groups to handle goroutine exit

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
				atomic.SwapUint64(&r.reqCount, 0)
				atomic.SwapUint64(&r.resCount, 0) // TODO: send these to a channel to update UI
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

func (r *Runner) LoadTest() {
	r.getRequestSpec()

	r.validateRequests()

	duration := time.Duration(r.config.Duration) * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	r.ctx = ctx

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	r.getRPS(ticker)

	r.initCounters()

	for i := 0; i < r.config.NumClients; i++ {
		client := newClient(r.requests, r.ctx, r.wg, r.reqCountChan, r.resCountChan)
		client.start()
	}

	r.wg.Wait()

	log.Printf("INFO:  error count: %d\n", r.errorCount)
}
