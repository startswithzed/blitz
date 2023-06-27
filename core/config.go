package core

import "time"

type Config struct {
	ReqSpecPath     string
	Duration        time.Duration
	NumClients      int
	MetricsEndpoint string
}
