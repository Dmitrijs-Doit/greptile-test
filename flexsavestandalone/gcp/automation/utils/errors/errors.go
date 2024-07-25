package errors

func NewFirstIterationError() *FirstIterationError {
	return &FirstIterationError{}
}

type FirstIterationError struct {
}

func (f *FirstIterationError) Error() string {
	return "skipping due to been first iteration"
}
