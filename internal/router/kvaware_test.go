package router

import (
	"context"
	"errors"
	"testing"
	"time"
)

type stubStore struct {
	backend string
	ok      bool
	err     error
}

func (s stubStore) PreferredBackend(context.Context, string) (string, bool, error) {
	return s.backend, s.ok, s.err
}

func (s stubStore) RememberBackend(context.Context, string, string, time.Duration) error {
	return nil
}

func TestKVAwarePrefersCachedBackend(t *testing.T) {
	a, _ := NewBackend("a", "http://a.test")
	b, _ := NewBackend("b", "http://b.test")

	strategy := NewKVAware([]*Backend{a, b}, stubStore{backend: "b", ok: true})
	got, err := strategy.Select(SelectionInput{
		Context:  context.Background(),
		CacheKey: "prefix",
	})
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	defer got.Release()

	if got.Backend.Name != "b" {
		t.Fatalf("expected cached backend b, got %s", got.Backend.Name)
	}
}

func TestKVAwareFallsBackWhenStoreErrors(t *testing.T) {
	a, _ := NewBackend("a", "http://a.test")
	b, _ := NewBackend("b", "http://b.test")

	strategy := NewKVAware([]*Backend{a, b}, stubStore{err: errors.New("redis down")})
	first, err := strategy.Select(SelectionInput{
		Context:  context.Background(),
		CacheKey: "prefix",
	})
	if err != nil {
		t.Fatalf("select first: %v", err)
	}
	second, err := strategy.Select(SelectionInput{
		Context:  context.Background(),
		CacheKey: "prefix",
	})
	if err != nil {
		t.Fatalf("select second: %v", err)
	}
	defer first.Release()
	defer second.Release()

	if first.Backend.Name == second.Backend.Name {
		t.Fatalf("expected least-pending fallback to spread load, got %s twice", first.Backend.Name)
	}
}

func TestKVAwareSkipsUnhealthyCachedBackend(t *testing.T) {
	a, _ := NewBackend("a", "http://a.test")
	b, _ := NewBackend("b", "http://b.test")
	b.SetHealthy(false)

	strategy := NewKVAware([]*Backend{a, b}, stubStore{backend: "b", ok: true})
	got, err := strategy.Select(SelectionInput{
		Context:  context.Background(),
		CacheKey: "prefix",
	})
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	defer got.Release()

	if got.Backend.Name != "a" {
		t.Fatalf("expected healthy fallback backend a, got %s", got.Backend.Name)
	}
}
