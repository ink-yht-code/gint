package discovery

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestServiceInstanceAddr(t *testing.T) {
	tests := []struct {
		name     string
		instance ServiceInstance
		want     string
	}{
		{
			name:     "with port",
			instance: ServiceInstance{Address: "127.0.0.1", Port: 8080},
			want:     "127.0.0.1:8080",
		},
		{
			name:     "without port",
			instance: ServiceInstance{Address: "service.local", Port: 0},
			want:     "service.local",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.instance.Addr(); got != tc.want {
				t.Fatalf("Addr() = %s, want %s", got, tc.want)
			}
			if got := tc.instance.FullAddr(); got != tc.want {
				t.Fatalf("FullAddr() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestMemoryRegistryConcurrentRegisterAndGet(t *testing.T) {
	r := NewMemoryRegistry(DefaultConfig())
	ctx := context.Background()

	const n = 100
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			inst := &ServiceInstance{
				ID:      fmt.Sprintf("inst-%d", i),
				Name:    "user-service",
				Address: "127.0.0.1",
				Port:    8000 + i,
				Weight:  1,
			}
			if err := r.Register(ctx, inst); err != nil {
				t.Errorf("Register() error: %v", err)
			}
		}()
	}

	wg.Wait()

	instances, err := r.GetInstances(ctx, "user-service")
	if err != nil {
		t.Fatalf("GetInstances() error: %v", err)
	}
	if len(instances) != n {
		t.Fatalf("len(instances) = %d, want %d", len(instances), n)
	}
}

func TestMemoryRegistryTimeout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.HeartbeatTimeout = 10 * time.Millisecond
	r := NewMemoryRegistry(cfg)
	ctx := context.Background()

	inst := &ServiceInstance{
		ID:      "inst-1",
		Name:    "order-service",
		Address: "127.0.0.1",
		Port:    9000,
	}

	if err := r.Register(ctx, inst); err != nil {
		t.Fatalf("Register() error: %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	_, err := r.GetInstances(ctx, "order-service")
	if err == nil {
		t.Fatal("expected error after heartbeat timeout, got nil")
	}
	if err != ErrNoInstance {
		t.Fatalf("GetInstances() error = %v, want %v", err, ErrNoInstance)
	}
}
