package core

import (
	"bytes"
	"context"
	"log"
	"math/rand"
	"net/http"
	"sync"
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
	ctx          context.Context
	wg           *sync.WaitGroup
	reqCountChan chan<- struct{}
	resCountChan chan<- struct{}
	errorStream  chan<- ErrorLog
}

func newClient(
	reqs []*Request,
	ctx context.Context,
	wg *sync.WaitGroup,
	reqCountChan chan struct{},
	resCountChan chan struct{},
) *client {
	return &client{
		requests:     reqs,
		ctx:          ctx,
		wg:           wg,
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
	rand.Seed(time.Now().UnixNano())
	c.wg.Add(1)

	go func(ctx context.Context) {
		defer c.wg.Done()

		for {
			select {
			case <-ctx.Done():
				log.Println("INFO: shutting down client goroutine")
				return
			default:
				request := c.requests[rand.Intn(len(c.requests))]

				resp, err := c.sendRequest(request)
				if err != nil {
					log.Println(err) // TODO: return this error in an error stream
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
		}
	}(c.ctx)
}
