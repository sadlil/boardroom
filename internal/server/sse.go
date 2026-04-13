package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/golang/glog"
	"github.com/sadlil/boardroom/ui"
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
	err := h.orchestrator.RunDebate(ctx, session.Prompt, session.UseDynamicAgents, func(agentID, chunk string) {
		sendEvent(agentID, chunk)
	})

	if err != nil {
		glog.Errorf("Debate pipeline failed for session %s: %v\n", sessionID, err)
		h.sessions.SetStatus(sessionID, "error")

		// Render the fatal error banner and send it via a special "error" SSE event
		errorHTML, renderErr := ui.RenderToString("toast_fatal.html", struct {
			Message string
		}{Message: err.Error()})
		if renderErr != nil {
			errorHTML = fmt.Sprintf("<div class='text-rose-400 p-4'>Critical error: %s</div>", err.Error())
		}

		mu.Lock()
		encodedError, _ := json.Marshal(errorHTML)
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", string(encodedError))
		flusher.Flush()
		mu.Unlock()
		return
	}

	h.sessions.MarkCompleted(sessionID)
	
	// Save outputs to SQLite
	sessionData := h.sessions.Get(sessionID)
	if sessionData != nil {
		sessionData.mu.RLock()
		for role, content := range sessionData.AgentOutputs {
			h.sqlite.SaveSessionLog(sessionID, role, content)
		}
		sessionData.mu.RUnlock()
	}

	glog.Infof("SSE stream completed for session %s (data preserved in memory & sqlite)\n", sessionID)
}
