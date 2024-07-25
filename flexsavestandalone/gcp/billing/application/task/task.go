package task

type Task interface {
	Init() error
	Verify() error
	Run() error
	TearDown() error
}
