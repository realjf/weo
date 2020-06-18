package stream

func New() *Basic {
	return &Basic{
		StopCh: make(chan struct{}),
	}
}

type Basic struct {
	StopCh chan struct{}
	Error error
}

func (s *Basic) Close() error {
	close(s.StopCh)
	return nil
}

func (s *Basic) Err() error {
	return s.Error
}


