package main

// TODO: Implement graceful shutdown by handling interrupt signals and using context and wait groups

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

type Request struct {
	Verb    string            `json:"verb"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    interface{}       `json:"body"`
}

type Response struct {
	StatusCode   int
	ResponseTime int64
	Timestamp    int64
}

func validateRequests(requests []Request) {
	// TODO: check only GET, POST, PUT and DELETE are present

	// TODO: handle body parsing errors and replace body with byte stream
}

func sendRequest(request *Request, reqCountChan chan<- struct{}, resCountChan chan<- struct{}) (Response, error) {
	client := &http.Client{}

	var req *http.Request
	var resp *http.Response
	var err error
	var startTime time.Time
	var body []byte

	switch request.Verb {
	case "GET":
		req, err = http.NewRequest(request.Verb, request.URL, nil)
		for k, v := range request.Headers {
			req.Header.Set(k, v)
		}
	case "POST":
		body, err = json.Marshal(request.Body) // TODO: Handle body parsing errors in the validate function and store byte slice instead
		req, err = http.NewRequest(request.Verb, request.URL, bytes.NewReader(body))
		for k, v := range request.Headers {
			req.Header.Set(k, v)
		}
	case "PUT":
		body, err = json.Marshal(request.Body)
		req, err = http.NewRequest(request.Verb, request.URL, bytes.NewReader(body))
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

func ProcessResponse(response Response) {
	//fmt.Println("INFO:  ", response.Timestamp, response.StatusCode, response.ResponseTime)
}

func main() {
	var err error

	reqSpec, err := os.Open("request_spec.json")
	defer reqSpec.Close()

	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("Successfully Opened %s\n", reqSpec.Name())

	bytes, err := io.ReadAll(reqSpec)
	if err != nil {
		fmt.Println(err)
	}

	var requests []*Request

	err = json.Unmarshal(bytes, &requests)
	if err != nil {
		fmt.Println(err)
	}

	//validateRequests(requests)

	timeout := 10 * time.Second
	timeoutChan := time.After(timeout)

	numClients := 10

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
			fmt.Printf("INFO:  requests per second: %d, responses per second: %d\n", request, response)
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
					fmt.Println(err)
				}

				if resp.StatusCode >= 300 || resp.StatusCode < 200 {
					fmt.Printf("Error:  timestamp: %d\t verb: %s\turl: %s\tstatus code: %d\n", resp.Timestamp, request.Verb, request.URL, resp.StatusCode)
					atomic.AddUint64(&errorCount, 1)
				}

			}
		}(requests)
	}

	select {
	case <-timeoutChan:
		fmt.Println("INFO:  Test duration completed. Ending test")
	}

	fmt.Printf("INFO:  error count: %d\n", errorCount)
}
