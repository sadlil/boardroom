package openai

import (
	"context"

	"github.com/sadlil/boardroom/internal/llm"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

type Client struct {
	model llms.Model
}

func NewClient(apiKey, modelName string) (*Client, error) {
	if modelName == "" {
		modelName = "gpt-4o"
	}
	model, err := openai.New(
		openai.WithToken(apiKey),
		openai.WithModel(modelName),
	)
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
