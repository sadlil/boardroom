package server

import (
	"context"
	"net/http"
	"time"

	"github.com/sadlil/boardroom/internal/agents"
	"github.com/sadlil/boardroom/internal/database"
)

// Server wraps the HTTP server with graceful lifecycle management.
type Server struct {
	httpServer *http.Server
}

// NewServer creates and wires up the HTTP server with all routes.
func NewServer(sqlite *database.SQLiteDB, memory *database.VectorMemory, orchestrator *agents.Orchestrator) *Server {
	mux := http.NewServeMux()

	h := &Handler{
		sqlite:       sqlite,
		memory:       memory,
		orchestrator: orchestrator,
		sessions:     NewSessionStore(),
	}
	h.Register(mux)

	return &Server{
		httpServer: &http.Server{
			Addr:              ":8080",
			Handler:           SecurityMiddleware(mux),
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      0, // Disabled: SSE streams are long-lived; LLM debates can run for minutes
			IdleTimeout:       120 * time.Second,
		},
	}
}

// ListenAndServe starts listening for HTTP connections.
func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the server with a 5-second timeout.
func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}
