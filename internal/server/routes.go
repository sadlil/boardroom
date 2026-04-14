package server

import (
	"net/http"

	"github.com/sadlil/boardroom/internal/agents"
	"github.com/sadlil/boardroom/internal/database"
	"github.com/sadlil/boardroom/ui"
)

// Handler holds dependencies for all HTTP handlers.
type Handler struct {
	sqlite       *database.SQLiteDB
	memory       *database.VectorMemory
	orchestrator *agents.Orchestrator
	sessions     *SessionStore
}

// Register mounts all routes onto the given ServeMux.
func (h *Handler) Register(mux *http.ServeMux) {
	// Static assets
	mux.Handle("GET /", http.FileServer(ui.Assets))

	// API endpoints
	mux.HandleFunc("GET /api/init", h.handleInit)
	mux.HandleFunc("POST /api/onboard", h.handleOnboard)
	mux.HandleFunc("GET /api/config", h.handleConfig)
	mux.HandleFunc("GET /api/session", h.handleGetSession)
	mux.HandleFunc("GET /api/history", h.handleGetHistory)
	mux.HandleFunc("GET /api/memories", h.handleGetMemories)
	mux.HandleFunc("POST /api/memories", h.handleUpdateMemories)
	mux.HandleFunc("POST /api/debate", h.handleStartDebate)
	mux.HandleFunc("POST /api/cancel", h.handleCancelDebate)
	mux.HandleFunc("DELETE /api/session", h.handleDeleteSession)
	mux.HandleFunc("GET /api/stream", h.handleSSEStream)
}
