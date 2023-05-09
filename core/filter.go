package core

type Filter interface {
	Init() error
	Filter(e ...*Event) ([]*Event)
	Close() error
}
