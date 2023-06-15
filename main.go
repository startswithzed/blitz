package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
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

	// TODO: convert verb from string to Verb

	// TODO: check if body exists for post and put request then the content type is json

	// TODO: handle body parsing errors and replace body with byte stream
}

func sendRequest(request *Request) (Response, error) {
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

	startTime = time.Now()
	resp, err = client.Do(req)
	if err != nil {
		return Response{}, err
	}

	responseTime := time.Since(startTime)

	return Response{
		StatusCode:   resp.StatusCode,
		ResponseTime: responseTime.Milliseconds(),
		Timestamp:    startTime.UnixNano(),
	}, nil
}

func startClient(requests []*Request, timeoutChan <-chan time.Time) {
	fmt.Println("INFO:  Client started")
	defer fmt.Println("INFO:  Shutting Down Goroutine")
	for {
		select {
		case <-timeoutChan:
			fmt.Printf("INFO:  Timeout occurred\n")
		default:
			rand.Seed(time.Now().UnixNano())
			request := requests[rand.Intn(len(requests))]
			resp, err := sendRequest(request)
			if err != nil {
				fmt.Println(err)
			}

			fmt.Println(resp.StatusCode, resp.ResponseTime)
		}
	}
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

	timeout := 5 * time.Second
	timeoutChan := time.After(timeout)

	numWorkers := 5
	for i := 0; i < numWorkers; i++ {
		go startClient(requests, timeoutChan)
	}

	select {
	case <-timeoutChan:
		fmt.Println("INFO:  Test duration completed. Ending test")
	}
}
