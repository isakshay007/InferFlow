package router

import "sync"

const (
	StrategyRoundRobin   = "round_robin"
	StrategyLeastPending = "least_pending"
	StrategyCostAware    = "cost_aware"
)

type Strategy interface {
	Name() string
	SetBackends(backends []*Backend)
	Select(estimatedCost int) (Decision, error)
	HasHealthyBackend() bool
}

type Decision struct {
	Backend         *Backend
	PendingRequests int64
	PendingCost     int64

	release func()
	once    sync.Once
}

func (d *Decision) Release() {
	d.once.Do(func() {
		if d.release != nil {
			d.release()
		}
	})
}
