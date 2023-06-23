package core

import (
	"bytes"
	"fmt"
	"math/rand"
	"net/http"
	"time"
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

type client struct {
	requests     []*Request
	reqCountChan chan<- struct{}
	resCountChan chan<- struct{}
	errorStream  chan<- ErrorLog
}

func newClient(reqs []*Request, reqCountChan chan struct{}, resCountChan chan struct{}) *client {
	return &client{
		requests:     reqs,
		reqCountChan: reqCountChan,
		resCountChan: resCountChan,
	}
}

func (c *client) sendRequest(request *Request) (Response, error) {
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

	startTime = time.Now()
	c.reqCountChan <- struct{}{}
	resp, err = client.Do(req)
	if err != nil {
		return Response{}, err
	}

	responseTime := time.Since(startTime)
	c.resCountChan <- struct{}{}

	return Response{
		StatusCode:   resp.StatusCode,
		ResponseTime: responseTime.Milliseconds(),
		Timestamp:    startTime.UnixNano(),
	}, nil
}

func (c *client) start() {
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			request := c.requests[rand.Intn(len(c.requests))]

			resp, err := c.sendRequest(request)
			if err != nil {
				fmt.Println(err) // TODO: return this error in an error stream
			}

			if resp.StatusCode >= 300 || resp.StatusCode < 200 {
				c.errorStream <- ErrorLog{
					Timestamp:  resp.Timestamp,
					Verb:       request.Verb,
					URL:        request.URL,
					StatusCode: resp.StatusCode,
				}
				continue
			}
		}
	}()
}
