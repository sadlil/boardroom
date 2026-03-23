package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

// handleSSEStream establishes an SSE connection and runs the debate pipeline.
func (h *Handler) handleSSEStream(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("id")
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	session := h.sessions.Get(sessionID)
	if session == nil {
		http.Error(w, "Invalid session", http.StatusBadRequest)
		return
	}

	if session.Status == "completed" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	var mu sync.Mutex

	sendEvent := func(agentID, chunk string) {
		mu.Lock()
		defer mu.Unlock()

		if ctx.Err() != nil {
			return
		}

		h.sessions.AppendOutput(sessionID, agentID, chunk)

		encodedChunk, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", agentID, string(encodedChunk))
		flusher.Flush()
	}

	// Run the orchestrator synchronously to guarantee we don't return from
	// the HTTP handler before all goroutines inside have fully shut down.
	h.orchestrator.RunDebate(ctx, session.Prompt, session.UseDynamicAgents, func(agentID, chunk string) {
		sendEvent(agentID, chunk)
	})

	h.sessions.MarkCompleted(sessionID)
	log.Printf("SSE stream completed for session %s (data preserved in memory)", sessionID)
}
