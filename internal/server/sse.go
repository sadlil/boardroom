package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/golang/glog"
	"github.com/sadlil/boardroom/ui"
)

// handleSSEStream establishes an SSE connection and tails the session event log.
// The debate pipeline runs independently in a background goroutine (started by
// handleStartDebate), so disconnecting here does NOT cancel the debate.
// Reconnection is supported via the SSE Last-Event-Id header.
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

	// If the session is already completed with no more events to stream,
	// return 204 so the EventSource closes cleanly.
	session.mu.RLock()
	status := session.Status
	eventCount := len(session.Events)
	session.mu.RUnlock()

	// Determine the starting read position.
	// Priority: Last-Event-Id header (native reconnect) > "from" query param (session resume) > 0
	fromIdx := 0
	if lastID := r.Header.Get("Last-Event-Id"); lastID != "" {
		if parsed, err := strconv.Atoi(lastID); err == nil && parsed >= 0 {
			fromIdx = parsed
		}
	} else if fromParam := r.URL.Query().Get("from"); fromParam != "" {
		if parsed, err := strconv.Atoi(fromParam); err == nil && parsed >= 0 {
			fromIdx = parsed
		}
	}

	// If already completed and the client has caught up, nothing to send
	if (status == "completed" || status == "error" || status == "cancelled") && fromIdx >= eventCount {
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

	glog.Infof("SSE client connected for session %s (resuming from event %d)", sessionID, fromIdx)

	// Tail the event log until the debate finishes or the client disconnects
	for {
		events, nextIdx, done := h.sessions.WaitForEvents(sessionID, fromIdx, r.Context())

		// Client disconnected — just exit, the pipeline keeps running
		if r.Context().Err() != nil {
			glog.Infof("SSE client disconnected for session %s at event %d (debate continues in background)", sessionID, fromIdx)
			return
		}

		// Write all new events to the SSE stream
		for i, evt := range events {
			encodedChunk, _ := json.Marshal(evt.Chunk)
			eventIdx := fromIdx + i
			fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", eventIdx+1, evt.AgentID, string(encodedChunk))
		}
		flusher.Flush()
		fromIdx = nextIdx

		if done {
			glog.Infof("SSE stream completed for session %s (%d events total)", sessionID, fromIdx)
			return
		}
	}
}

// runDebateBackground runs the debate pipeline in a background goroutine
// with a non-request-bound context. The pipeline stores events in the session
// event log, and SSE handlers independently tail them.
func (h *Handler) runDebateBackground(sessionID string) {
	session := h.sessions.Get(sessionID)
	if session == nil {
		return
	}

	// Use a non-request-bound context so the debate survives browser disconnects
	ctx, cancel := context.WithCancel(context.Background())

	session.mu.Lock()
	session.Cancel = cancel
	session.mu.Unlock()

	defer cancel()

	glog.Infof("Starting background debate pipeline for session %s", sessionID)

	err := h.orchestrator.RunDebate(ctx, session.Prompt, session.UseDynamicAgents, func(agentID, chunk string) {
		h.sessions.AppendEvent(sessionID, agentID, chunk)
	})

	if err != nil {
		glog.Errorf("Debate pipeline failed for session %s: %v", sessionID, err)
		h.sessions.SetStatus(sessionID, "error")

		// Render the fatal error banner and push it as an event
		errorHTML, renderErr := ui.RenderToString("toast_fatal.html", struct {
			Message string
		}{Message: err.Error()})
		if renderErr != nil {
			errorHTML = fmt.Sprintf("<div class='text-rose-400 p-4'>Critical error: %s</div>", err.Error())
		}
		h.sessions.AppendEvent(sessionID, "error", errorHTML)
		return
	}

	h.sessions.MarkCompleted(sessionID)

	// Persist all agent outputs to SQLite
	session.mu.RLock()
	outputs := session.GetOutputs()
	session.mu.RUnlock()

	for role, content := range outputs {
		if err := h.sqlite.SaveSessionLog(sessionID, role, content); err != nil {
			glog.Errorf("Failed to save session log for %s/%s: %v", sessionID, role, err)
		}
	}

	glog.Infof("Background debate pipeline completed for session %s (persisted to SQLite)", sessionID)
}
