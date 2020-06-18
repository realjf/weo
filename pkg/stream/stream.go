package stream

type Stream interface {
	Close() error
	Err() error
}
