package llm

import (
	"context"
)

// Message represents a single conversational turn
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// StreamProcessor is a callback function invoked each time a new token/chunk is received from an LLM.
type StreamProcessor func(chunk string) error

// Client unified interface for LLM streaming.
type Client interface {
	// StreamCompletion takes a system prompt, a slice of previous messages, and a callback processor,
	// and blocks until the stream completes or the context is cancelled.
	StreamCompletion(ctx context.Context, systemPrompt string, messages []Message, processor StreamProcessor) error
}
