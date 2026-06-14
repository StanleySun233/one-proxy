package sla

import (
	"context"
	"testing"
	"time"
)

func TestMemoryHeartbeatStoreCountsDistinctSlots(t *testing.T) {
	store := NewMemoryHeartbeatStore(time.Hour, 10*time.Second)
	start := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	if err := store.Record(context.Background(), "node-1", start.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := store.Record(context.Background(), "node-1", start.Add(9*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := store.Record(context.Background(), "node-1", start.Add(10*time.Second)); err != nil {
		t.Fatal(err)
	}
	count, err := store.Count(context.Background(), "node-1", start, start.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
}

func TestMemoryHeartbeatStoreRetention(t *testing.T) {
	store := NewMemoryHeartbeatStore(time.Minute, 10*time.Second)
	start := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	if err := store.Record(context.Background(), "node-1", start); err != nil {
		t.Fatal(err)
	}
	if err := store.Record(context.Background(), "node-1", start.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	count, err := store.Count(context.Background(), "node-1", start, start.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}
