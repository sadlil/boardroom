package agents

import (
	"context"
	"strings"
	"time"

	"github.com/sadlil/boardroom/internal/llm"
	"github.com/sadlil/boardroom/ui"
)

// executeWithRetry runs an LLM call with exponential-backoff retries and
// toast notifications on failure.
func (o *Orchestrator) executeWithRetry(ctx context.Context, agentID string, agentName string, systemPrompt string, messages []llm.Message, callback StreamCallback) (string, error) {
	maxRetries := 3
	backoff := 5 * time.Second

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		var fullResponse strings.Builder

		if attempt > 1 {
			msg, _ := ui.RenderToString("toast_retry.html", struct {
				ToastID    int64
				AgentName  string
				Backoff    time.Duration
				Attempt    int
				MaxRetries int
			}{ToastID: time.Now().UnixNano(), AgentName: agentName, Backoff: backoff, Attempt: attempt, MaxRetries: maxRetries})

			callback("toast", msg)

			// Use context-aware sleep to abort if user cancels request
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return "", ctx.Err()
			}
			backoff *= 2
		}

		err := o.llmClient.StreamCompletion(ctx, systemPrompt, messages, func(chunk string) error {
			fullResponse.WriteString(chunk)
			if agentID != "" && agentID != "clarification" { // Only stream UI panels, ignoring invisible checks
				callback(agentID, chunk)
			}
			return nil
		})

		if err == nil {
			return fullResponse.String(), nil
		}

		lastErr = err
		if ctx.Err() != nil {
			return "", err
		}
	}

	errMsg, _ := ui.RenderToString("toast_error.html", struct {
		ToastID    int64
		AgentName  string
		MaxRetries int
	}{ToastID: time.Now().UnixNano(), AgentName: agentName, MaxRetries: maxRetries})

	callback("toast", errMsg)
	return "", lastErr
}
