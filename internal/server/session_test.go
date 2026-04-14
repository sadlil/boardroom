package server

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestSessionStore(t *testing.T) {
	os.Setenv("MAX_SESSIONS", "2") // Set max size to trigger eviction trivially
	defer os.Unsetenv("MAX_SESSIONS")

	store := NewSessionStore()

	// Test Create
	id1 := store.Create("prompt 1", false)
	if id1 == "" {
		t.Fatal("Expected non-empty session ID")
	}

	// Test Get
	session := store.Get(id1)
	if session == nil {
		t.Fatalf("Session %s not found", id1)
	}
	if session.Prompt != "prompt 1" {
		t.Errorf("Expected prompt 'prompt 1', got %s", session.Prompt)
	}

	// Test AppendEvent (streams into event log + builders)
	store.AppendEvent(id1, "agent1", "hello ")
	store.AppendEvent(id1, "agent1", "world")

	// Verify event log
	session.mu.RLock()
	if len(session.Events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(session.Events))
	}
	session.mu.RUnlock()

	// Verify builder accumulation
	outputs := session.GetOutputs()
	if outputs["agent1"] != "hello world" {
		t.Errorf("Expected builder output 'hello world', got '%s'", outputs["agent1"])
	}

	// Test WaitForEvents — should return immediately when events exist
	ctx := context.Background()
	events, nextIdx, done := store.WaitForEvents(id1, 0, ctx)
	if len(events) != 2 {
		t.Errorf("Expected 2 events from WaitForEvents, got %d", len(events))
	}
	if nextIdx != 2 {
		t.Errorf("Expected nextIdx=2, got %d", nextIdx)
	}
	if done {
		t.Error("Expected done=false for running session")
	}

	// No new events beyond position 2 — WaitForEvents would block,
	// so verify with a cancelled context
	cancelCtx, cancelFn := context.WithCancel(ctx)
	cancelFn()
	events2, nextIdx2, done2 := store.WaitForEvents(id1, 2, cancelCtx)
	if len(events2) != 0 {
		t.Errorf("Expected 0 events for caught-up reader, got %d", len(events2))
	}
	if nextIdx2 != 2 {
		t.Errorf("Expected nextIdx=2, got %d", nextIdx2)
	}
	if !done2 {
		t.Error("Expected done=true when context cancelled")
	}

	// Test MarkCompleted
	store.MarkCompleted(id1)
	if session.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", session.Status)
	}

	// WaitForEvents should return done=true after completion
	events3, _, done3 := store.WaitForEvents(id1, 2, ctx)
	if len(events3) != 0 {
		t.Errorf("Expected 0 new events after completion, got %d", len(events3))
	}
	if !done3 {
		t.Error("Expected done=true for completed session")
	}

	// Wait briefly to ensure unique timestamps for next creates if needed, though UnixNano usually fine.
	time.Sleep(1 * time.Millisecond)

	// Test Eviction Logic (MAX_SESSIONS is 2)
	id2 := store.Create("prompt 2", true)

	// We have 3 items created, but cache size is 2. id1 should be evicted if id2 and id3 are newest
	// But wait, Get(id1) was called, which might promote it? Let's check eviction.
	// We'll just create a 4th one to be absolutely certain the older ones get dropped.
	id4 := store.Create("prompt 4", false)

	if store.Get(id1) != nil && store.Get(id2) != nil {
		t.Errorf("Expected at least one session to be evicted, but id1 and id2 still exist")
	}

	if store.Get(id4) == nil {
		t.Errorf("Expected newest session id4 to exist")
	}
}

func TestControlEventsSkipBuilder(t *testing.T) {
	store := NewSessionStore()
	id := store.Create("test", false)

	// Toast and dynamic_panel events should NOT accumulate in builders
	store.AppendEvent(id, "toast", "<div>retry</div>")
	store.AppendEvent(id, "dynamic_panel", "<div>panel</div>")
	store.AppendEvent(id, "agent1", "real content")

	session := store.Get(id)
	outputs := session.GetOutputs()

	if _, exists := outputs["toast"]; exists {
		t.Error("toast events should not be in builder outputs")
	}
	if _, exists := outputs["dynamic_panel"]; exists {
		t.Error("dynamic_panel events should not be in builder outputs")
	}
	if outputs["agent1"] != "real content" {
		t.Errorf("Expected 'real content', got '%s'", outputs["agent1"])
	}

	// But all 3 events should be in the event log
	session.mu.RLock()
	if len(session.Events) != 3 {
		t.Errorf("Expected 3 events in log, got %d", len(session.Events))
	}
	session.mu.RUnlock()
}
