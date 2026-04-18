package router

import "testing"

func TestCostAwareDistributesAcrossHealthyBackendsOnEqualCost(t *testing.T) {
	a, _ := NewBackend("a", "http://a.test")
	b, _ := NewBackend("b", "http://b.test")
	ca := NewCostAware([]*Backend{a, b})

	first, err := ca.Select(SelectionInput{EstimatedCost: 10})
	if err != nil {
		t.Fatalf("select first: %v", err)
	}
	second, err := ca.Select(SelectionInput{EstimatedCost: 10})
	if err != nil {
		t.Fatalf("select second: %v", err)
	}
	defer first.Release()
	defer second.Release()

	if first.Backend.Name == second.Backend.Name {
		t.Fatalf("expected different backends, got %s twice", first.Backend.Name)
	}
}

func TestCostAwareRoutesLongPromptsAwayFromLoadedBackend(t *testing.T) {
	a, _ := NewBackend("a", "http://a.test")
	b, _ := NewBackend("b", "http://b.test")
	ca := NewCostAware([]*Backend{a, b})

	heavy, err := ca.Select(SelectionInput{EstimatedCost: 120})
	if err != nil {
		t.Fatalf("select heavy: %v", err)
	}

	light, err := ca.Select(SelectionInput{EstimatedCost: 10})
	if err != nil {
		t.Fatalf("select light: %v", err)
	}
	if heavy.Backend.Name == light.Backend.Name {
		t.Fatalf("expected heavy request to push next request to other backend")
	}

	light.Release()
	longPrompt, err := ca.Select(SelectionInput{EstimatedCost: 200})
	if err != nil {
		t.Fatalf("select long prompt: %v", err)
	}
	defer heavy.Release()
	defer longPrompt.Release()

	if longPrompt.Backend.Name != light.Backend.Name {
		t.Fatalf("expected long prompt on lower-cost backend %s, got %s", light.Backend.Name, longPrompt.Backend.Name)
	}
}

func TestCostAwareSkipsUnhealthyBackends(t *testing.T) {
	a, _ := NewBackend("a", "http://a.test")
	b, _ := NewBackend("b", "http://b.test")
	a.SetHealthy(false)
	ca := NewCostAware([]*Backend{a, b})

	got, err := ca.Select(SelectionInput{EstimatedCost: 10})
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	defer got.Release()

	if got.Backend.Name != "b" {
		t.Fatalf("expected backend b, got %s", got.Backend.Name)
	}
}

func TestCostAwareReturnsErrorWhenNoneHealthy(t *testing.T) {
	a, _ := NewBackend("a", "http://a.test")
	a.SetHealthy(false)
	ca := NewCostAware([]*Backend{a})

	if _, err := ca.Select(SelectionInput{EstimatedCost: 10}); err != ErrNoHealthyBackend {
		t.Fatalf("expected ErrNoHealthyBackend, got %v", err)
	}
}
