package pkg

import (
	"sync"
	"testing"
)

func TestGetStatus(t *testing.T) {
	// Reset for test
	statusMutex.Lock()
	Status = Start
	statusMutex.Unlock()

	if got := GetStatus(); got != Start {
		t.Errorf("GetStatus() = %v, want %v", got, Start)
	}
}

func TestSetStatus(t *testing.T) {
	// Drain any leftover messages from previous tests.
	drainChan()

	// Reset
	statusMutex.Lock()
	Status = Start
	statusMutex.Unlock()

	// Set to Init and verify channel notification
	go func() {
		SetStatus(Init)
	}()
	got := <-StatusUpdating
	if got != Init {
		t.Errorf("expected Init on channel, got %v", got)
	}
	if GetStatus() != Init {
		t.Errorf("GetStatus() = %v, want Init", GetStatus())
	}
}

func TestSetStatus_SameStatus(t *testing.T) {
	drainChan()

	statusMutex.Lock()
	Status = Healthy
	statusMutex.Unlock()

	// Setting the same status should be a no-op (no channel send)
	done := make(chan struct{})
	go func() {
		SetStatus(Healthy)
		close(done)
	}()
	<-done // Should return immediately without blocking
}

func TestSetStatus_Concurrent(t *testing.T) {
	drainChan()

	statusMutex.Lock()
	Status = Start
	statusMutex.Unlock()

	// Consume status updates in background
	var received []NodeStatus
	var mu sync.Mutex
	done := make(chan struct{})
	go func() {
		for i := 0; i < 3; i++ {
			s := <-StatusUpdating
			mu.Lock()
			received = append(received, s)
			mu.Unlock()
		}
		close(done)
	}()

	// Set 3 different statuses sequentially (each blocks until consumed)
	SetStatus(Init)
	SetStatus(Healthy)
	SetStatus(Unhealthy)

	<-done

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 3 {
		t.Errorf("expected 3 status updates, got %d", len(received))
	}
}

func TestNodeStatus_ToString(t *testing.T) {
	tests := []struct {
		status NodeStatus
		want   string
	}{
		{Start, "start"},
		{Init, "init"},
		{Healthy, "healthy"},
		{Unhealthy, "unhealthy"},
		{Rebalancing, "rebalancing"},
		{Inactive, "inactive"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.ToString(); got != tt.want {
				t.Errorf("ToString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func drainChan() {
	for {
		select {
		case <-StatusUpdating:
		default:
			return
		}
	}
}
