package agents

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/tools"
	"github.com/tmc/langchaingo/tools/duckduckgo"
)

// SearchTool implements an internet search tool utilizing DuckDuckGo
// to gather free, localized search results without needing an API key.
type SearchTool struct {
	ddg *duckduckgo.Tool
}

var _ tools.Tool = &SearchTool{}

// Name returns the name of the tool.
func (t *SearchTool) Name() string {
	return "InternetSearch"
}

// Description returns a description of the tool.
func (t *SearchTool) Description() string {
	return "Useful for searching the internet for the latest news, market data, and facts."
}

// Call executes the tool.
func (t *SearchTool) Call(ctx context.Context, input string) (string, error) {
	fmt.Printf("[SearchTool] Agent invoked search query: %s\n", input)

	// Lazy initialize the DuckDuckGo tool if not done
	if t.ddg == nil {
		var err error
		// 5 results max to keep prompt context short
		t.ddg, err = duckduckgo.New(5, duckduckgo.DefaultUserAgent)
		if err != nil {
			return "", fmt.Errorf("failed to initialize DuckDuckGo tool: %w", err)
		}
	}

	result, err := t.ddg.Call(ctx, input)
	if err != nil {
		fmt.Printf("[SearchTool] Error searching: %v\n", err)
		return "", fmt.Errorf("search failed")
	}

	return result, nil
}
