package agents

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/sadlil/boardroom/internal/database"
	"github.com/sadlil/boardroom/internal/llm/fake"
)

func TestOrchestrator_RunDebate(t *testing.T) {
	// Setup isolated dependencies
	client := fake.NewClient()

	sqliteDBPath := filepath.Join(t.TempDir(), "test.db")
	sqlite, err := database.NewSQLiteDB(sqliteDBPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite: %v", err)
	}
	defer sqlite.Close()

	vectorDBPath := filepath.Join(t.TempDir(), "vector")
	// Mock embedding for offline integration test
	mockEmbed := func(ctx context.Context, text string) ([]float32, error) {
		return []float32{0.1, 0.2, 0.3}, nil
	}
	memory, err := database.NewVectorMemory(vectorDBPath, mockEmbed)
	if err != nil {
		t.Fatalf("Failed to create VectorMemory: %v", err)
	}

	// Chromem-go requires nResults to be <= collection size
	// We must populate exactly 5 documents if the orchestrator natively requests 5.
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		memory.AddDocument(ctx, fmt.Sprintf("dummy-%d", i), "Dummy Context Data", nil)
	}

	orchestrator := NewOrchestrator(client, sqlite, memory)

	// Stream capture
	var mu sync.Mutex
	capturedStream := make(map[string]strings.Builder)
	callback := func(agentID, chunk string) {
		mu.Lock()
		defer mu.Unlock()
		builder := capturedStream[agentID]
		builder.WriteString(chunk)
		capturedStream[agentID] = builder
	}

	// Run Debate End-to-End
	prompt := "Should we transition to microservices?"

	// Execute without dynamic agents for stability
	orchestrator.RunDebate(ctx, prompt, false, callback)

	// Since fake.Client naturally returns mocked string payloads for StreamCompletion,
	// we should see output captured for standard debaters.

	mu.Lock()
	defer mu.Unlock()

	// Verify that multiple agents participated out of the execution waves.
	// We expect the Decider and some Context or Debaters to have streamed tokens.
	if len(capturedStream) == 0 {
		t.Fatalf("Expected callback to capture streaming tokens, got empty map")
	}

	// Verify decider output streamed
	deciderStream, ok := capturedStream[Decider.ID]
	if !ok || len(deciderStream.String()) == 0 {
		t.Errorf("Expected Decider (%s) to stream output, got none", Decider.ID)
	}
}
