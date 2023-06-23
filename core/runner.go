package core

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

type Runner struct {
	config       Config
	requests     []*Request
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
		reqCountChan: make(chan struct{}, config.NumClients),
		resCountChan: make(chan struct{}, config.NumClients),
		errorStream:  make(chan ErrorLog),
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
	// TODO: use context api to gracefully exit goroutines

	go func() {
		for range r.reqCountChan {
			atomic.AddUint64(&r.reqCount, 1)
		}
	}()

	go func() {
		for range r.resCountChan {
			atomic.AddUint64(&r.resCount, 1)
		}
	}()
}

func (r *Runner) getRPS(ticker *time.Ticker) {
	// TODO: add graceful exit logic
	go func() {
		for range ticker.C {
			request := atomic.SwapUint64(&r.reqCount, 0)
			response := atomic.SwapUint64(&r.resCount, 0)
			log.Printf("INFO:  requests per second: %d, responses per second: %d\n", request, response)
		}
	}()
}

func (r *Runner) countErrors() {
	go func() {
		for range r.errorStream {
			atomic.AddUint64(&r.errorCount, 1)
		}
	}()
}

func (r *Runner) LoadTest() {
	// TODO: end test after duration

	r.getRequestSpec()

	r.validateRequests()

	r.initCounters()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	r.getRPS(ticker)

	duration := time.Duration(r.config.Duration) * time.Minute
	timeoutChan := time.After(duration)

	for i := 0; i < r.config.NumClients; i++ {
		client := newClient(r.requests, r.reqCountChan, r.resCountChan)
		client.start()
	}

	select {
	case <-timeoutChan:
		log.Println("INFO:  Test duration completed. Ending test")
	}

	log.Printf("INFO:  error count: %d\n", r.errorCount)
}
