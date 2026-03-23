package ollama

import (
	"context"
	"os"

	"github.com/sadlil/boardroom/internal/llm"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

type Client struct {
	model llms.Model
}

func NewClient() (*Client, error) {
	modelName := os.Getenv("LLM_MODEL")
	if modelName == "" {
		modelName = "gemma3:1b"
	}

	opts := []ollama.Option{
		ollama.WithModel(modelName),
	}

	host := os.Getenv("OLLAMA_HOST")
	if host != "" {
		opts = append(opts, ollama.WithServerURL(host))
	}

	model, err := ollama.New(opts...)
	if err != nil {
		return nil, err
	}

	return &Client{model: model}, nil
}

func (c *Client) StreamCompletion(ctx context.Context, systemPrompt string, messages []llm.Message, processor llm.StreamProcessor) error {
	var content []llms.MessageContent

	if systemPrompt != "" {
		content = append(content, llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt))
	}

	for _, msg := range messages {
		roleType := llms.ChatMessageTypeHuman
		switch msg.Role {
		case "assistant":
			roleType = llms.ChatMessageTypeAI
		case "system":
			roleType = llms.ChatMessageTypeSystem
		}

		content = append(content, llms.TextParts(roleType, msg.Content))
	}

	_, err := c.model.GenerateContent(ctx, content, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		return processor(string(chunk))
	}))

	return err
}
