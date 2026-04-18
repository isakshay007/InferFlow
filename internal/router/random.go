package router

import (
	"math/rand"
	"sync"
)

type Random struct {
	backends []*Backend
	mu       sync.RWMutex
	rng      *rand.Rand
}

func NewRandom(backends []*Backend) *Random {
	return &Random{
		backends: backends,
		rng:      rand.New(rand.NewSource(1)),
	}
}

func (r *Random) Name() string {
	return StrategyRandom
}

func (r *Random) SetBackends(backends []*Backend) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.backends = backends
}

func (r *Random) Select(_ SelectionInput) (Decision, error) {
	r.mu.RLock()
	backends := append([]*Backend(nil), r.backends...)
	r.mu.RUnlock()

	healthy := make([]*Backend, 0, len(backends))
	for _, backend := range backends {
		if backend.Healthy() {
			healthy = append(healthy, backend)
		}
	}
	if len(healthy) == 0 {
		return Decision{}, ErrNoHealthyBackend
	}

	r.mu.Lock()
	idx := r.rng.Intn(len(healthy))
	r.mu.Unlock()

	return Decision{Backend: healthy[idx]}, nil
}

func (r *Random) HasHealthyBackend() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, backend := range r.backends {
		if backend.Healthy() {
			return true
		}
	}
	return false
}
