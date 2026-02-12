package multicluster

import (
	"testing"
	"time"
)

func TestClientPool_GetMissing(t *testing.T) {
	pool := &ClientPool{
		clients: make(map[string]*ClusterClient),
	}

	_, err := pool.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing cluster")
	}
}

func TestClientPool_GetOpen(t *testing.T) {
	cb := NewCircuitBreaker(1, 30*time.Second)
	cb.RecordFailure() // opens the breaker

	pool := &ClientPool{
		clients: map[string]*ClusterClient{
			"test": {
				Name:           "test",
				CircuitBreaker: cb,
			},
		},
	}

	_, err := pool.Get("test")
	if err == nil {
		t.Fatal("expected error for open circuit breaker")
	}
}

func TestClientPool_GetHealthy(t *testing.T) {
	cb := NewCircuitBreaker(3, 30*time.Second)

	pool := &ClientPool{
		clients: map[string]*ClusterClient{
			"test": {
				Name:           "test",
				CircuitBreaker: cb,
			},
		},
	}

	cc, err := pool.Get("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cc.Name != "test" {
		t.Fatalf("expected name 'test', got %q", cc.Name)
	}
}

func TestClientPool_ListAndNames(t *testing.T) {
	pool := &ClientPool{
		clients: map[string]*ClusterClient{
			"alpha": {Name: "alpha"},
			"beta":  {Name: "beta"},
		},
	}

	list := pool.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 clients, got %d", len(list))
	}

	names := pool.Names()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
}
