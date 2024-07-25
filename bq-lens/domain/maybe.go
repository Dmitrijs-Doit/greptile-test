package domain

type Maybe[T any] struct {
	Value T
	Err   error
}
