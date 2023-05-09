package core

type Processor interface {
	Init() error
	Process(e ...*Event) []*Event
	Close() error
}
