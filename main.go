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

func ProcessResponse(response Response) {
	fmt.Println("INFO:  ", response.Timestamp, response.StatusCode, response.ResponseTime)
}

func Funnel(sources ...chan Response) <-chan Response {
	output := make(chan Response) // TODO: Close this channel and goroutines using wg

	for _, source := range sources {
		go func(c <-chan Response) {
			for {
				select {
				case resp, ok := <-c:
					if !ok {
						return
					}
					output <- resp
				}
			}
		}(source)
	}

	return output
}

func Split(source <-chan Response, numWorkers int) {
	for i := 0; i < numWorkers; i++ {
		go func() {
			for {
				select {
				case resp, ok := <-source:
					if !ok {
						return
					}
					ProcessResponse(resp)
				}
			}
		}()
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

	timeout := 10 * time.Second
	timeoutChan := time.After(timeout)

	numClients := 10
	numWorkers := 5

	var sources []chan Response

	for i := 0; i < numClients; i++ {
		source := make(chan Response)
		sources = append(sources, source)
	}

	// spawn clients and send requests
	for i := 0; i < numClients; i++ {
		go func(requests []*Request, source chan<- Response) {
			for {
				rand.Seed(time.Now().UnixNano())
				request := requests[rand.Intn(len(requests))]

				resp, err := sendRequest(request)
				if err != nil {
					fmt.Println(err)
				}

				source <- resp
			}
		}(requests, sources[i])
	}

	// collect responses
	resChan := Funnel(sources...) // TODO: check the directionality of the channels

	// spawn workers and process responses
	Split(resChan, numWorkers)

	select {
	case <-timeoutChan:
		fmt.Println("INFO:  Test duration completed. Ending test")
	}
}
