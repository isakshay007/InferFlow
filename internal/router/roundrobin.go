package router

import (
	"sync"
	"sync/atomic"
)

type RoundRobin struct {
	backends []*Backend
	next     atomic.Uint64
	mu       sync.RWMutex
}

func NewRoundRobin(backends []*Backend) *RoundRobin {
	return &RoundRobin{backends: backends}
}

func (r *RoundRobin) SetBackends(backends []*Backend) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.backends = backends
}

func (r *RoundRobin) Name() string {
	return StrategyRoundRobin
}

func (r *RoundRobin) Pick() (*Backend, error) {
	r.mu.RLock()
	backends := append([]*Backend(nil), r.backends...)
	r.mu.RUnlock()

	if len(backends) == 0 {
		return nil, ErrNoHealthyBackend
	}

	start := int(r.next.Add(1)-1) % len(backends)
	for i := 0; i < len(backends); i++ {
		backend := backends[(start+i)%len(backends)]
		if backend.Healthy() {
			return backend, nil
		}
	}

	return nil, ErrNoHealthyBackend
}

func (r *RoundRobin) Select(_ SelectionInput) (Decision, error) {
	backend, err := r.Pick()
	if err != nil {
		return Decision{}, err
	}
	return Decision{Backend: backend}, nil
}

func (r *RoundRobin) HasHealthyBackend() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, backend := range r.backends {
		if backend.Healthy() {
			return true
		}
	}
	return false
}
