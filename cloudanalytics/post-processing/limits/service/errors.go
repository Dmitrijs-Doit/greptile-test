package service

import "errors"

var (
	ErrIndexOutOfBounds = errors.New("index out of bounds")
	ErrInvalidType      = errors.New("invalid type")
)
