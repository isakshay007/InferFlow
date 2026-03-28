package router

import (
	"sync"
	"sync/atomic"
)

type CostAware struct {
	backends []*Backend
	next     atomic.Uint64
	mu       sync.RWMutex
	loads    map[*Backend]*atomic.Int64
}

func NewCostAware(backends []*Backend) *CostAware {
	s := &CostAware{}
	s.SetBackends(backends)
	return s
}

func (s *CostAware) Name() string {
	return StrategyCostAware
}

func (s *CostAware) SetBackends(backends []*Backend) {
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

func (s *CostAware) Select(estimatedCost int) (Decision, error) {
	s.mu.RLock()
	backends := append([]*Backend(nil), s.backends...)
	loads := s.loads
	s.mu.RUnlock()

	if len(backends) == 0 {
		return Decision{}, ErrNoHealthyBackend
	}

	cost := int64(estimatedCost)
	if cost < 1 {
		cost = 1
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

	pendingAfter := chosenLoad.Add(cost)
	return Decision{
		Backend:     chosen,
		PendingCost: pendingAfter,
		release: func() {
			chosenLoad.Add(-cost)
		},
	}, nil
}

func (s *CostAware) HasHealthyBackend() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, backend := range s.backends {
		if backend.Healthy() {
			return true
		}
	}
	return false
}
