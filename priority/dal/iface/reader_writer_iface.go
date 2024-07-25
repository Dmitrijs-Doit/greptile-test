//go:generate mockery --output=../mocks --name=ReaderWriter --filename=reader_writer_iface.go
package iface

type ReaderWriter interface {
	Reader
	Writer
}
