package common

import "context"

type Syncer struct {
	lastDone chan struct{}
}

// NewSyncer creates a new Syncer.
func NewSyncer() *Syncer {
	prevDone := make(chan struct{}, 1)
	prevDone <- struct{}{}

	return &Syncer{
		lastDone: prevDone,
	}
}

func (s *Syncer) WorkerSyncer() *WorkerSyncer {
	prevDone := s.lastDone
	s.lastDone = make(chan struct{}, 1)

	return &WorkerSyncer{
		start: prevDone,
		done:  s.lastDone,
	}
}

type WorkerSyncer struct {
	start chan struct{}
	done  chan struct{}
}

func (s *WorkerSyncer) WaitPrevious(ctx context.Context) {
	select {
	case <-s.start:
	case <-ctx.Done():
	}
}

func (s *WorkerSyncer) Done(ctx context.Context) {
	select {
	case s.done <- struct{}{}:
	case <-ctx.Done():
	}
}
