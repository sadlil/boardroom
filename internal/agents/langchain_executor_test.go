package agents

import (
	"context"
	"strings"
	"testing"

	"github.com/sadlil/boardroom/internal/llm"
	"github.com/tmc/langchaingo/tools"
)

// mockLLMClient is a specialized fake LLM just for triggering tools.
type mockToolTriggerClient struct {
	triggerCount int
}

func (m *mockToolTriggerClient) StreamCompletion(ctx context.Context, systemPrompt string, messages []llm.Message, processor llm.StreamProcessor) error {
	var resp string

	// Force the LLM to trigger a tool on the first pass
	if m.triggerCount == 0 {
		m.triggerCount++
		resp = "TOOL_CALL: MockSearch What is the capital of France?"
	} else {
		// Second pass handles the injected observation
		lastMsg := messages[len(messages)-1].Content
		if strings.Contains(lastMsg, "System Update: The 'MockSearch' tool was executed") {
			resp = "FINAL: The capital is Paris."
		} else {
			resp = "FINAL: I couldn't figure it out."
		}
	}

	return processor(resp)
}

type MockSearch struct {
	called bool
}

func (m *MockSearch) Name() string        { return "MockSearch" }
func (m *MockSearch) Description() string { return "Mock tool" }
func (m *MockSearch) Call(ctx context.Context, input string) (string, error) {
	m.called = true
	return "Mock result for " + input, nil
}

func TestReActExecutor(t *testing.T) {
	mockClient := &mockToolTriggerClient{}

	orchestrator := NewOrchestrator(mockClient, nil, nil)
	mockTool := &MockSearch{}

	agentTools := []tools.Tool{mockTool}
	agent := Agent{
		ID:           "test",
		Name:         "Test",
		SystemPrompt: "Test Prompt",
	}

	result, err := orchestrator.ReActExecutor(context.Background(), agent, "Ask something", agentTools, func(id, chunk string) {})
	if err != nil {
		t.Fatalf("ReActExecutor failed: %v", err)
	}

	if !mockTool.called {
		t.Fatal("Expected tool to be invoked by the ReActExecutor, but it was not called")
	}

	if !strings.Contains(result, "Paris") {
		t.Errorf("Expected final response to contain 'Paris', got %v", result)
	}
}
