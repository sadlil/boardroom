package server

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"

	lru "github.com/hashicorp/golang-lru/v2"
)

// SSEEvent represents a single event in the debate stream.
type SSEEvent struct {
	AgentID string
	Chunk   string
}

// SessionData holds the state for a single debate session.
type SessionData struct {
	mu               sync.RWMutex // Protects Events, AgentBuilders, Status, notify
	Cancel           context.CancelFunc
	Prompt           string
	Events           []SSEEvent                 // Append-only event log (the source of truth)
	AgentBuilders    map[string]*strings.Builder // Streaming buffers — avoids O(n²) string concat
	notify           chan struct{}               // Closed-and-replaced to broadcast new events
	Status           string                     // "running", "completed", "error", "cancelled"
	UseDynamicAgents bool
}

// GetOutputs materializes the accumulated text per agent from the builders,
// plus concatenates control events (toast, dynamic_panel) from the event log.
// This produces the same map[string]string shape needed by session_restore.html.
// Call under at least an RLock.
func (sd *SessionData) GetOutputs() map[string]string {
	out := make(map[string]string, len(sd.AgentBuilders)+2)
	for id, b := range sd.AgentBuilders {
		out[id] = b.String()
	}
	// Include control events that were excluded from builders
	for _, evt := range sd.Events {
		if evt.AgentID == "dynamic_panel" || evt.AgentID == "toast" {
			out[evt.AgentID] += evt.Chunk
		}
	}
	return out
}

// SessionStore wraps a hashicorp LRU cache with typed session accessors.
// When the cache exceeds its max size, the least recently used session is evicted.
type SessionStore struct {
	cache *lru.Cache[string, *SessionData]
}

// NewSessionStore creates an LRU session store.
// Max size is read from MAX_SESSIONS env var (default: 100).
func NewSessionStore() *SessionStore {
	maxSize := 100
	if envMax := os.Getenv("MAX_SESSIONS"); envMax != "" {
		if parsed, err := strconv.Atoi(envMax); err == nil && parsed > 0 {
			maxSize = parsed
		}
	}

	cache, _ := lru.New[string, *SessionData](maxSize)
	glog.Infof("Session store initialized (LRU max size: %d)", maxSize)

	return &SessionStore{cache: cache}
}

// Create initializes a new session and returns its ID.
func (s *SessionStore) Create(prompt string, useDynamicAgents bool) string {
	id := fmt.Sprintf("session_%d", time.Now().UnixNano())
	s.cache.Add(id, &SessionData{
		Prompt:           prompt,
		Events:           make([]SSEEvent, 0, 256),
		AgentBuilders:    make(map[string]*strings.Builder),
		notify:           make(chan struct{}),
		Status:           "running",
		UseDynamicAgents: useDynamicAgents,
	})
	return id
}

// Get retrieves a session by ID. Returns nil if not found or evicted.
// Accessing a session promotes it in the LRU order.
func (s *SessionStore) Get(id string) *SessionData {
	val, ok := s.cache.Get(id)
	if !ok {
		return nil
	}
	return val
}

// AppendEvent appends an SSE event to the session's event log and accumulates
// agent text in the streaming builders. Wakes up all waiting SSE readers.
func (s *SessionStore) AppendEvent(id, agentID, chunk string) {
	val, ok := s.cache.Get(id)
	if !ok {
		return
	}

	val.mu.Lock()
	defer val.mu.Unlock()

	// Append to the ordered event log
	val.Events = append(val.Events, SSEEvent{AgentID: agentID, Chunk: chunk})

	// Accumulate text in builders (skip control events that aren't agent prose)
	if agentID != "toast" && agentID != "dynamic_panel" && agentID != "error" {
		b, exists := val.AgentBuilders[agentID]
		if !exists {
			b = &strings.Builder{}
			val.AgentBuilders[agentID] = b
		}
		b.WriteString(chunk)
	}

	// Broadcast: close the current notify channel (wakes all waiters) and create a fresh one
	close(val.notify)
	val.notify = make(chan struct{})
}

// WaitForEvents blocks until new events are available beyond fromIdx, or the
// context is cancelled. Returns the new events, the next read index, and
// whether the stream is finished (completed/error/cancelled).
func (s *SessionStore) WaitForEvents(id string, fromIdx int, ctx context.Context) ([]SSEEvent, int, bool) {
	val, ok := s.cache.Get(id)
	if !ok {
		return nil, fromIdx, true
	}

	for {
		val.mu.RLock()
		eventLen := len(val.Events)
		status := val.Status
		notifyCh := val.notify
		val.mu.RUnlock()

		// If there are new events, return them
		if eventLen > fromIdx {
			val.mu.RLock()
			events := make([]SSEEvent, eventLen-fromIdx)
			copy(events, val.Events[fromIdx:eventLen])
			val.mu.RUnlock()

			done := status == "completed" || status == "error" || status == "cancelled"
			return events, eventLen, done
		}

		// If the stream is finished and we've caught up, we're done
		if status == "completed" || status == "error" || status == "cancelled" {
			return nil, fromIdx, true
		}

		// Wait for new events or client disconnect
		select {
		case <-notifyCh:
			// New events available, loop back to read them
		case <-ctx.Done():
			// Client disconnected — return without killing the debate
			return nil, fromIdx, true
		}
	}
}

// MarkCompleted sets the session status to "completed" and wakes all waiters.
func (s *SessionStore) MarkCompleted(id string) {
	val, ok := s.cache.Get(id)
	if !ok {
		return
	}

	val.mu.Lock()
	val.Status = "completed"
	close(val.notify)
	val.notify = make(chan struct{})
	val.mu.Unlock()
}

// SetStatus sets the session status and wakes all waiters.
func (s *SessionStore) SetStatus(id, status string) {
	val, ok := s.cache.Get(id)
	if !ok {
		return
	}

	val.mu.Lock()
	val.Status = status
	close(val.notify)
	val.notify = make(chan struct{})
	val.mu.Unlock()
}
