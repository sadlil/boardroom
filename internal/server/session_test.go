package server

import (
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

	// Test AppendOutput
	store.AppendOutput(id1, "agent1", "hello ")
	store.AppendOutput(id1, "agent1", "world")

	if session.AgentOutputs["agent1"] != "hello world" {
		t.Errorf("Expected appended output 'hello world', got '%s'", session.AgentOutputs["agent1"])
	}

	// Test Status Update
	store.MarkCompleted(id1)
	if session.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", session.Status)
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
