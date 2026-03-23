package server

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

// SessionData holds the state for a single debate session.
type SessionData struct {
	Prompt           string
	AgentOutputs     map[string]string
	Status           string // "running" or "completed"
	UseDynamicAgents bool
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
	log.Printf("Session store initialized (LRU max size: %d)", maxSize)

	return &SessionStore{cache: cache}
}

// Create initializes a new session and returns its ID.
func (s *SessionStore) Create(prompt string, useDynamicAgents bool) string {
	id := fmt.Sprintf("session_%d", time.Now().UnixNano())
	s.cache.Add(id, &SessionData{
		Prompt:           prompt,
		AgentOutputs:     make(map[string]string),
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

// AppendOutput appends a chunk to the given agent's output buffer.
func (s *SessionStore) AppendOutput(id, agentID, chunk string) {
	val, ok := s.cache.Get(id)
	if !ok {
		return
	}
	if val.AgentOutputs == nil {
		val.AgentOutputs = make(map[string]string)
	}
	val.AgentOutputs[agentID] += chunk
}

// MarkCompleted sets the session status to "completed".
func (s *SessionStore) MarkCompleted(id string) {
	val, ok := s.cache.Get(id)
	if !ok {
		return
	}
	val.Status = "completed"
}
