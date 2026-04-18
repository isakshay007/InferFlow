package router

import (
	"sync"
	"sync/atomic"
)

type LeastPending struct {
	backends []*Backend
	next     atomic.Uint64
	mu       sync.RWMutex
	loads    map[*Backend]*atomic.Int64
}

func NewLeastPending(backends []*Backend) *LeastPending {
	s := &LeastPending{}
	s.SetBackends(backends)
	return s
}

func (s *LeastPending) Name() string {
	return StrategyLeastPending
}

func (s *LeastPending) SetBackends(backends []*Backend) {
	s.mu.Lock()
	defer s.mu.Unlock()

	updated := make(map[*Backend]*atomic.Int64, len(backends))
	for _, backend := range backends {
		if existing, ok := s.loads[backend]; ok {
			updated[backend] = existing
			continue
		}
		updated[backend] = &atomic.Int64{}
	}
	s.backends = backends
	s.loads = updated
}

func (s *LeastPending) Select(_ SelectionInput) (Decision, error) {
	s.mu.RLock()
	backends := append([]*Backend(nil), s.backends...)
	loads := s.loads
	s.mu.RUnlock()

	if len(backends) == 0 {
		return Decision{}, ErrNoHealthyBackend
	}

	start := int(s.next.Add(1)-1) % len(backends)

	var chosen *Backend
	var chosenLoad *atomic.Int64
	var minPending int64

	for i := 0; i < len(backends); i++ {
		backend := backends[(start+i)%len(backends)]
		if !backend.Healthy() {
			continue
		}
		load, ok := loads[backend]
		if !ok {
			continue
		}
		pending := load.Load()
		if chosen == nil || pending < minPending {
			chosen = backend
			chosenLoad = load
			minPending = pending
		}
	}

	if chosen == nil || chosenLoad == nil {
		return Decision{}, ErrNoHealthyBackend
	}

	pendingAfter := chosenLoad.Add(1)
	return Decision{
		Backend:         chosen,
		PendingRequests: pendingAfter,
		release: func() {
			chosenLoad.Add(-1)
		},
	}, nil
}

func (s *LeastPending) HasHealthyBackend() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, backend := range s.backends {
		if backend.Healthy() {
			return true
		}
	}
	return false
}
