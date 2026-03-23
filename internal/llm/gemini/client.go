package gemini

import (
	"context"

	"github.com/sadlil/boardroom/internal/llm"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/googleai"
)

type Client struct {
	model llms.Model
}

func NewClient(apiKey, modelName string) (*Client, error) {
	if modelName == "" {
		modelName = "gemini-2.5-flash"
	}
	model, err := googleai.New(context.Background(), googleai.WithAPIKey(apiKey), googleai.WithDefaultModel(modelName))
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
		if msg.Role == "assistant" {
			roleType = llms.ChatMessageTypeAI
		} else if msg.Role == "system" {
			roleType = llms.ChatMessageTypeSystem
		}

		content = append(content, llms.TextParts(roleType, msg.Content))
	}

	_, err := c.model.GenerateContent(ctx, content, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		return processor(string(chunk))
	}))

	return err
}
