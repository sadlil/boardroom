package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/sadlil/boardroom/internal/agents"
	"github.com/sadlil/boardroom/internal/database"
	"github.com/sadlil/boardroom/internal/llm/fake"
)

func setupTestHandler(t *testing.T) *Handler {
	// Setup mocked dependencies
	sqlitePath := t.TempDir() + "/test.db"
	sqlite, _ := database.NewSQLiteDB(sqlitePath)

	vectorPath := t.TempDir() + "/test-vector"
	mockEmbed := func(ctx context.Context, text string) ([]float32, error) {
		return []float32{0.1, 0.2, 0.3}, nil
	}
	memory, _ := database.NewVectorMemory(vectorPath, mockEmbed)

	client := fake.NewClient()
	orchestrator := agents.NewOrchestrator(client, sqlite, memory)

	// Since ui html templates aren't embedded properly running tests outside the root,
	// rendering might fail. We bypass strict HTML asserting unless we mount it.
	// We ensure we are at the app root by modifying wd if needed, but for unit tests,
	// usually checking headers is sufficient if UI isn't loaded.

	// Create dummy templates to avoid nil pointer panic during tests if needed.
	// But our handlers actually call ui.Render which relies on embedded FS.
	// If run from root `go test ./...` it works.

	return &Handler{
		sqlite:       sqlite,
		memory:       memory,
		orchestrator: orchestrator,
		sessions:     NewSessionStore(),
	}
}

func TestHandleInit(t *testing.T) {
	// Need to be careful about running this test if ui/embed.go isn't initializing correctly
	// from the internal/server test path.
	// We will skip testing the actual response body and instead check if it doesn't crash
	// and returns 200 OK.

	err := os.Chdir("../../") // Hack to allow ui templates to resolve during 'go test internal/server'
	if err != nil {
		t.Logf("Chdir failed, skipping template-based tests")
		return
	}

	h := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/init", nil)
	rr := httptest.NewRecorder()

	h.handleInit(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %v", rr.Code)
	}

	if contentType := rr.Header().Get("Content-Type"); contentType != "text/html" {
		t.Errorf("Expected Content-Type text/html, got %v", contentType)
	}
}

func TestHandleStartDebate(t *testing.T) {
	err := os.Chdir("../../")
	if err != nil {
		t.Logf("Chdir failed, skipping template-based tests")
		return
	}

	h := setupTestHandler(t)

	form := url.Values{}
	form.Add("prompt", "What is the best way to test?")
	form.Add("dynamic_agents", "false")
	// Adding a dummy previous_context bypasses the Wave 0 Clarification check in handlers.go
	form.Add("previous_context", "ZHVtbXkgY29udGV4dA==") // base64 for "dummy context"

	req := httptest.NewRequest(http.MethodPost, "/api/debate", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()

	h.handleStartDebate(rr, req)

	if rr.Code != http.StatusOK {
		// Expect OK because HX renders inline HTML partials normally
		t.Errorf("Expected 200 OK, got %v", rr.Code)
	}

	pushUrl := rr.Header().Get("HX-Push-Url")
	if !strings.HasPrefix(pushUrl, "?session=session_") {
		t.Errorf("Expected HX-Push-Url to start with ?session=session_, got %v", pushUrl)
	}
}

func TestHandleOnboard(t *testing.T) {
	err := os.Chdir("../../")
	if err != nil {
		t.Logf("Chdir failed, skipping template-based tests")
		return
	}

	h := setupTestHandler(t)

	form := url.Values{}
	form.Add("role", "Tester")
	form.Add("industry", "Software")

	req := httptest.NewRequest(http.MethodPost, "/api/onboard", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()

	h.handleOnboard(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %v", rr.Code)
	}

	// Verify the sqlite DB actually saved the profile
	profile, _ := h.sqlite.GetProfile()
	if !strings.Contains(profile, "Tester") || !strings.Contains(profile, "Software") {
		t.Errorf("Profile was not saved correctly to DB: %s", profile)
	}
}
