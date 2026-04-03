package loadbalance

import (
	"fmt"
	"sync"
	"testing"

	"github.com/ink-yht-code/gint/discovery"
)

func TestWeightedRoundRobinDistribution(t *testing.T) {
	lb := NewWeightedRoundRobin()
	instances := []*discovery.ServiceInstance{
		{ID: "a", Weight: 5},
		{ID: "b", Weight: 1},
	}

	count := map[string]int{"a": 0, "b": 0}
	for i := 0; i < 60; i++ {
		inst, err := lb.Select(instances)
		if err != nil {
			t.Fatalf("Select() error: %v", err)
		}
		count[inst.ID]++
	}

	if count["a"] != 50 || count["b"] != 10 {
		t.Fatalf("unexpected distribution: a=%d b=%d, want a=50 b=10", count["a"], count["b"])
	}
}

func TestWeightedRoundRobinStateCleanup(t *testing.T) {
	lb := NewWeightedRoundRobin()
	instances := []*discovery.ServiceInstance{
		{ID: "a", Weight: 3},
		{ID: "b", Weight: 1},
	}

	for i := 0; i < 10; i++ {
		if _, err := lb.Select(instances); err != nil {
			t.Fatalf("Select() error: %v", err)
		}
	}

	onlyB := []*discovery.ServiceInstance{{ID: "b", Weight: 1}}
	for i := 0; i < 5; i++ {
		inst, err := lb.Select(onlyB)
		if err != nil {
			t.Fatalf("Select() with single instance error: %v", err)
		}
		if inst.ID != "b" {
			t.Fatalf("Select() got %s, want b", inst.ID)
		}
	}
}

func TestConsistentHashSelectByKeyStable(t *testing.T) {
	lb := NewConsistentHash()
	instances := []*discovery.ServiceInstance{
		{ID: "node-1"},
		{ID: "node-2"},
		{ID: "node-3"},
	}

	first, err := lb.SelectByKey(instances, "tenant:1001")
	if err != nil {
		t.Fatalf("SelectByKey() error: %v", err)
	}

	for i := 0; i < 20; i++ {
		cur, err := lb.SelectByKey(instances, "tenant:1001")
		if err != nil {
			t.Fatalf("SelectByKey() error: %v", err)
		}
		if cur.ID != first.ID {
			t.Fatalf("SelectByKey() unstable: first=%s current=%s", first.ID, cur.ID)
		}
	}
}

func TestConsistentHashSelectNotAlwaysFirst(t *testing.T) {
	lb := NewConsistentHash()
	instances := []*discovery.ServiceInstance{
		{ID: "node-1"},
		{ID: "node-2"},
		{ID: "node-3"},
	}

	seen := map[string]struct{}{}
	for i := 0; i < 50; i++ {
		inst, err := lb.Select(instances)
		if err != nil {
			t.Fatalf("Select() error: %v", err)
		}
		seen[inst.ID] = struct{}{}
	}

	if len(seen) < 2 {
		t.Fatalf("Select() seems degenerated, selected instances: %v", seen)
	}
}

func TestRandomConcurrentSelect(t *testing.T) {
	lb := NewRandom()
	instances := []*discovery.ServiceInstance{
		{ID: "node-1"},
		{ID: "node-2"},
		{ID: "node-3"},
	}

	errCh := make(chan error, 100)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			inst, err := lb.Select(instances)
			if err != nil {
				errCh <- fmt.Errorf("Select() error: %w", err)
				return
			}
			if inst == nil {
				errCh <- fmt.Errorf("Select() returned nil instance")
			}
		}()
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Error(err)
	}
}
