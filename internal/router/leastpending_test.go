package router

import "testing"

func TestLeastPendingDistributesAcrossHealthyBackends(t *testing.T) {
	a, _ := NewBackend("a", "http://a.test")
	b, _ := NewBackend("b", "http://b.test")
	lp := NewLeastPending([]*Backend{a, b})

	first, err := lp.Select(SelectionInput{EstimatedCost: 1})
	if err != nil {
		t.Fatalf("select first: %v", err)
	}
	second, err := lp.Select(SelectionInput{EstimatedCost: 1})
	if err != nil {
		t.Fatalf("select second: %v", err)
	}
	defer first.Release()
	defer second.Release()

	if first.Backend.Name == second.Backend.Name {
		t.Fatalf("expected different backends, got %s twice", first.Backend.Name)
	}
}

func TestLeastPendingSkipsUnhealthyBackends(t *testing.T) {
	a, _ := NewBackend("a", "http://a.test")
	b, _ := NewBackend("b", "http://b.test")
	a.SetHealthy(false)
	lp := NewLeastPending([]*Backend{a, b})

	got, err := lp.Select(SelectionInput{EstimatedCost: 1})
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	defer got.Release()

	if got.Backend.Name != "b" {
		t.Fatalf("expected backend b, got %s", got.Backend.Name)
	}
}

func TestLeastPendingPrefersLessBusyBackend(t *testing.T) {
	a, _ := NewBackend("a", "http://a.test")
	b, _ := NewBackend("b", "http://b.test")
	lp := NewLeastPending([]*Backend{a, b})

	first, err := lp.Select(SelectionInput{EstimatedCost: 1})
	if err != nil {
		t.Fatalf("select first: %v", err)
	}
	second, err := lp.Select(SelectionInput{EstimatedCost: 1})
	if err != nil {
		t.Fatalf("select second: %v", err)
	}

	if first.Backend.Name == second.Backend.Name {
		t.Fatalf("expected first two selects to spread across backends")
	}

	second.Release()
	third, err := lp.Select(SelectionInput{EstimatedCost: 1})
	if err != nil {
		t.Fatalf("select third: %v", err)
	}
	defer first.Release()
	defer third.Release()

	if third.Backend.Name != second.Backend.Name {
		t.Fatalf("expected less-busy backend %s, got %s", second.Backend.Name, third.Backend.Name)
	}
}

func TestLeastPendingReturnsErrorWhenNoneHealthy(t *testing.T) {
	a, _ := NewBackend("a", "http://a.test")
	a.SetHealthy(false)
	lp := NewLeastPending([]*Backend{a})

	if _, err := lp.Select(SelectionInput{EstimatedCost: 1}); err != ErrNoHealthyBackend {
		t.Fatalf("expected ErrNoHealthyBackend, got %v", err)
	}
}
