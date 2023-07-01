package core

type ResponseError struct {
	Timestamp  int64
	Verb       string
	URL        string
	StatusCode int
}

type NetworkError struct {
	Timestamp int64
	Error     error
}
