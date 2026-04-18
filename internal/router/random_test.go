package router

import "testing"

func TestRandomSkipsUnhealthyBackends(t *testing.T) {
	a, _ := NewBackend("a", "http://a.test")
	b, _ := NewBackend("b", "http://b.test")
	a.SetHealthy(false)

	random := NewRandom([]*Backend{a, b})
	got, err := random.Select(SelectionInput{})
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if got.Backend.Name != "b" {
		t.Fatalf("expected backend b, got %s", got.Backend.Name)
	}
}

func TestRandomReturnsErrorWhenNoneHealthy(t *testing.T) {
	a, _ := NewBackend("a", "http://a.test")
	a.SetHealthy(false)

	random := NewRandom([]*Backend{a})
	if _, err := random.Select(SelectionInput{}); err != ErrNoHealthyBackend {
		t.Fatalf("expected ErrNoHealthyBackend, got %v", err)
	}
}

func TestRandomEventuallyChoosesMultipleBackends(t *testing.T) {
	a, _ := NewBackend("a", "http://a.test")
	b, _ := NewBackend("b", "http://b.test")

	random := NewRandom([]*Backend{a, b})
	seen := map[string]bool{}
	for range 10 {
		got, err := random.Select(SelectionInput{})
		if err != nil {
			t.Fatalf("select: %v", err)
		}
		seen[got.Backend.Name] = true
	}
	if len(seen) < 2 {
		t.Fatalf("expected random selection to hit both backends, got %+v", seen)
	}
}
