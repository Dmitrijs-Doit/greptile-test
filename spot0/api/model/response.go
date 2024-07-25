package model

type Response struct {
	Done         bool   `json:"done"`
	ErrorMessage string `json:"error,omitempty"`
}
