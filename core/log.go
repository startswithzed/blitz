package core

type ErrorLog struct {
	Timestamp  int64
	Verb       string
	URL        string
	StatusCode int
}
