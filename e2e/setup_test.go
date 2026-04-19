package e2e

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

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
func TestMain(m *testing.M) {
	code, err := runTests(m)
	if err != nil {
		fmt.Fprintf(os.Stderr, "E2E Setup Failed: %v\n", err)
		os.Exit(1)
	}
	os.Exit(code)
}

func runTests(m *testing.M) (int, error) {
	// 1. Traverse up to repository root so html templates load properly
	err := os.Chdir("..")
	if err != nil {
		return 0, fmt.Errorf("failed to chdir to project root: %w", err)
	}

	// 2. Setup mock dependencies
	tempDir, err := os.MkdirTemp("", "boardroom-e2e")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(tempDir)

	sqlitePath := filepath.Join(tempDir, "e2e.db")
	sqlite, err := database.NewSQLiteDB(sqlitePath)
	if err != nil {
		return 0, err
	}
	defer sqlite.Close()

	vectorPath := filepath.Join(tempDir, "e2e-vector")
	mockEmbed := func(ctx context.Context, text string) ([]float32, error) {
		return make([]float32, 384), nil
	}
	memory, err := database.NewVectorMemory(vectorPath, mockEmbed)
	if err != nil {
		return 0, err
	}

	llmClient := fake.NewClient()
	orchestrator := agents.NewOrchestrator(llmClient, sqlite, memory)

	router := server.NewTestMux(sqlite, memory, orchestrator)
	testServer = httptest.NewServer(router)
	defer testServer.Close()
	fmt.Printf("E2E Test Server started at: %s\n", testServer.URL)

	// 3. Launch go-rod headless browser with connection timeout
	fmt.Println("Connecting to browser (may download Chromium if missing)...")
	
	// Create a context with a 2-minute deadline for the browser connection/download
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	browser = rod.New().Context(ctx)
	if err := browser.Connect(); err != nil {
		return 0, fmt.Errorf("failed to connect to browser within 2m: %w", err)
	}
	defer browser.Close()
	fmt.Println("Browser connected successfully.")

	// Run tests
	return m.Run(), nil
}
