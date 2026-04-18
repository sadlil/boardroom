package e2e

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-rod/rod"
	"github.com/sadlil/boardroom/internal/agents"
	"github.com/sadlil/boardroom/internal/database"
	"github.com/sadlil/boardroom/internal/llm/fake"
	"github.com/sadlil/boardroom/internal/server"
)

var (
	testServer *httptest.Server
	browser    *rod.Browser
)

// TestMain acts as the orchestrator for all e2e tests.
func TestMain(m *testing.T) {
	// 1. Traverse up to repository root so html templates load properly
	// e2e tests run from `/e2e` directory, but UI templates are at relative `./ui`.
	err := os.Chdir("..")
	if err != nil {
		panic("Failed to chdir to project root: " + err.Error())
	}

	// 2. Setup mock dependencies
	tempDir, err := os.MkdirTemp("", "boardroom-e2e")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir)

	sqlitePath := filepath.Join(tempDir, "e2e.db")
	sqlite, err := database.NewSQLiteDB(sqlitePath)
	if err != nil {
		panic(err)
	}
	defer sqlite.Close()

	vectorPath := filepath.Join(tempDir, "e2e-vector")
	mockEmbed := func(ctx context.Context, text string) ([]float32, error) {
		return make([]float32, 384), nil
	}
	memory, err := database.NewVectorMemory(vectorPath, mockEmbed)
	if err != nil {
		panic(err)
	}

	// Use fake LLM client so tests are fast and free
	llmClient := fake.NewClient()
	orchestrator := agents.NewOrchestrator(llmClient, sqlite, memory)

	// Inject test handler using exposed test helper
	router := server.NewTestMux(sqlite, memory, orchestrator)
	
	// Start test server
	testServer = httptest.NewServer(router)
	defer testServer.Close()

	// 3. Launch go-rod headless browser
	// By default, it automatically downloads Chromium if it doesn't find it.
	browser = rod.New().MustConnect()
	defer browser.MustClose()

	// Run tests
	code := m.Run()

	os.Exit(code)
}
