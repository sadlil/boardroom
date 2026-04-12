package agents

import (
	"context"
	"fmt"
	"strings"

	"github.com/sadlil/boardroom/internal/llm"
	"github.com/tmc/langchaingo/tools"
)

// ReActExecutor wraps an LLM client with LangChain-style tool execution capabilities.
// It detects if the agent organically requested a tool using 'TOOL_CALL:' within its constraints.
func (o *Orchestrator) ReActExecutor(ctx context.Context, agent Agent, prompt string, agentTools []tools.Tool, callback StreamCallback) (string, error) {
	// If the agent has no tools, fall back to the standard stream
	if len(agentTools) == 0 {
		return o.executeWithRetry(ctx, agent.ID, agent.Name, agent.SystemPrompt, []llm.Message{{Role: "user", Content: prompt}}, callback)
	}

	// 1. Augment System Prompt
	toolDesc := buildToolDescription(agentTools)
	systemPrompt := agent.SystemPrompt + "\n\n" + toolDesc

	// 2. Initial Generation (Agent attempts to answer or calls a tool)
	output, err := o.executeWithRetry(ctx, agent.ID, agent.Name, systemPrompt, []llm.Message{{Role: "user", Content: prompt}}, callback)
	if err != nil {
		return "", err
	}

	// 3. Parse Tool Invocation
	if strings.Contains(output, "TOOL_CALL:") {
		var usedTool tools.Tool
		var query string
		for _, t := range agentTools {
			if strings.Contains(output, "TOOL_CALL: "+t.Name()) {
				usedTool = t
				idx := strings.Index(output, "TOOL_CALL: "+t.Name())
				query = strings.TrimSpace(output[idx+len("TOOL_CALL: "+t.Name()):])
				// query might contain newlines if the model didn't stop generating, so take only the immediate query line
				query = strings.Split(query, "\n")[0]
				break
			}
		}

		if usedTool != nil {
			callback(agent.ID, fmt.Sprintf("\n\n*>[System: Invoking %s for Context Data]*\n", usedTool.Name()))
			result, err := usedTool.Call(ctx, query)

			if err == nil {
				callback(agent.ID, fmt.Sprintf("\n*>[System: Context Retrieved (%d bytes)]*\n\n", len(result)))
				// Inject the observation organically
				newPrompt := prompt + fmt.Sprintf("\n\n[System Update: The '%s' tool was executed and returned the following data:\n%s\n\nContinue and provide your final complete evaluation.]", usedTool.Name(), result)

				// 4. Final Generation with new context
				return o.executeWithRetry(ctx, agent.ID, agent.Name, systemPrompt, []llm.Message{{Role: "user", Content: newPrompt}}, callback)
			} else {
				callback(agent.ID, fmt.Sprintf("\n*>[System: Tool Failed - %v]*\n\n", err))
			}
		}
	}

	return output, nil
}

func buildToolDescription(ts []tools.Tool) string {
	var sb strings.Builder
	sb.WriteString("You have access to external tools. If you require external data before providing your final evaluation, output ONLY the string 'TOOL_CALL: [tool_name] [query]' and stop generating. Do not provide any conversational text before or after the tool call.\n\nAvailable Tools:\n")
	for _, t := range ts {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", t.Name(), t.Description()))
	}
	return sb.String()
}
