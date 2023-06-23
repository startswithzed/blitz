package main

// TODO: Implement graceful shutdown by handling interrupt signals and using context and wait groups

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/spf13/cobra"
)

type Request struct {
	Verb      string            `json:"verb"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	Body      interface{}       `json:"body"`
	BodyBytes []byte
}

type Response struct {
	StatusCode   int
	ResponseTime int64
	Timestamp    int64
}

func validateRequests(requests []*Request) []*Request {
	validRequests := make([]*Request, 0)

	for _, req := range requests {
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

	log.Printf("INFO:  total requests: %d\t valid requests: %d\n", len(requests), len(validRequests))

	return validRequests
}

func sendRequest(request *Request, reqCountChan chan<- struct{}, resCountChan chan<- struct{}) (Response, error) {
	client := &http.Client{}

	var req *http.Request
	var resp *http.Response
	var err error
	var startTime time.Time

	switch request.Verb {
	case "GET":
		req, err = http.NewRequest(request.Verb, request.URL, nil)
		for k, v := range request.Headers {
			req.Header.Set(k, v)
		}
	case "POST":
		req, err = http.NewRequest(request.Verb, request.URL, bytes.NewReader(request.BodyBytes))
		for k, v := range request.Headers {
			req.Header.Set(k, v)
		}
	case "PUT":
		req, err = http.NewRequest(request.Verb, request.URL, bytes.NewReader(request.BodyBytes))
		for k, v := range request.Headers {
			req.Header.Set(k, v)
		}
	case "DELETE":
		req, err = http.NewRequest(request.Verb, request.URL, nil)
		for k, v := range request.Headers {
			req.Header.Set(k, v)
		}
	}

	if err != nil {
		return Response{}, err
	}

	// send req
	startTime = time.Now()
	reqCountChan <- struct{}{}
	resp, err = client.Do(req)
	if err != nil {
		return Response{}, err
	}

	responseTime := time.Since(startTime)
	resCountChan <- struct{}{}

	return Response{
		StatusCode:   resp.StatusCode,
		ResponseTime: responseTime.Milliseconds(),
		Timestamp:    startTime.UnixNano(),
	}, nil
}

type Config struct {
	ReqSpecPath     string
	Duration        int
	NumClients      int
	MetricsEndpoint string
}

var config Config

func init() {
	rootCmd.Flags().StringVarP(&config.ReqSpecPath, "req-spec", "r", "", "Path to the request specification json file")
	rootCmd.Flags().IntVarP(&config.Duration, "duration", "d", 1, "Duration of the test in minutes")
	rootCmd.Flags().IntVarP(&config.NumClients, "num-clients", "c", 1, "Number of concurrent clients sending requests to the server")
	rootCmd.Flags().StringVarP(&config.MetricsEndpoint, "metrics-endpoint", "m", "", "Server metrics endpoint (optional)")

	rootCmd.MarkFlagRequired("req-spec")
}

var rootCmd = &cobra.Command{
	// TODO: Add detailed documentation

	Use:  "webruckus --req-spec /path/to/spec.json --duration 60 --num-clients 10",
	Long: "Load test your web server.",
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Handle error by shutting down the program

		var err error

		ext := filepath.Ext(config.ReqSpecPath)
		if ext != ".json" {
			log.Fatal("Invalid file format. Expected a JSON file.")
		}

		reqSpec, err := os.Open(config.ReqSpecPath)
		if err != nil {
			log.Fatalf("Could not open file: %v", err)
		}

		bytes, err := io.ReadAll(reqSpec)
		if err != nil {
			log.Fatalf("Error reading file: %v", err)
		}

		err = reqSpec.Close()
		if err != nil {
			log.Println("WARNING:  could not close file reader.")
		}

		var requests []*Request

		err = json.Unmarshal(bytes, &requests)
		if err != nil {
			log.Fatalf("Error parsing json file: %v", err)
		}

		requests = validateRequests(requests)

		duration := time.Duration(config.Duration) * time.Minute
		timeoutChan := time.After(duration)

		numClients := config.NumClients

		reqCountChan := make(chan struct{}, numClients)
		resCountChan := make(chan struct{}, numClients)
		var reqCount, resCount, errorCount uint64

		// launch counter goroutines
		go func() {
			for range reqCountChan {
				atomic.AddUint64(&reqCount, 1)
			}
		}()

		go func() {
			for range resCountChan {
				atomic.AddUint64(&resCount, 1)
			}
		}()

		// get RPS values
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		go func() {
			for range ticker.C {
				request := atomic.SwapUint64(&reqCount, 0)
				response := atomic.SwapUint64(&resCount, 0)
				log.Printf("INFO:  requests per second: %d, responses per second: %d\n", request, response)
			}
		}()

		// spawn clients and send requests
		for i := 0; i < numClients; i++ {
			go func(requests []*Request) {
				for {
					rand.Seed(time.Now().UnixNano())
					request := requests[rand.Intn(len(requests))]

					resp, err := sendRequest(request, reqCountChan, resCountChan)
					if err != nil {
						fmt.Println(err) // TODO: Handle this error in a separate stream
					}

					if resp.StatusCode >= 300 || resp.StatusCode < 200 {
						log.Printf("Error: timestamp: %d\t verb: %s\turl: %s\tstatus code: %d\n", resp.Timestamp, request.Verb, request.URL, resp.StatusCode)
						atomic.AddUint64(&errorCount, 1)
					}

				}
			}(requests)
		}

		select {
		case <-timeoutChan:
			log.Println("INFO:  Test duration completed. Ending test")
		}

		log.Printf("INFO:  error count: %d\n", errorCount)

	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
