package agents

import (
	"encoding/json"
	"fmt"
)

// Agent defines a board member with its identity and behavior.
type Agent struct {
	ID           string // Unique identifier used for SSE streaming (e.g. "optimist")
	Name         string // Display name (e.g. "Chief Visionary Officer")
	Persona      string // Short role label (e.g. "Optimist")
	Wave         string // Which wave this agent belongs to: "context", "debater", "decider", "dynamic"
	SystemPrompt string // The system instruction sent to the LLM
}

// StreamCallback is used to send tokens back to the UI via SSE.
type StreamCallback func(agentID string, chunk string)

// ContextPayload is the inter-agent protocol payload passed between waves.
type ContextPayload struct {
	UserPersona    string `json:"user_persona"`
	Macroeconomics string `json:"macroeconomics"`
}

// ClarificationResult holds the output of the Wave 0 clarification check.
type ClarificationResult struct {
	NeedsContext bool     `json:"needs_context"`
	Questions    []string `json:"questions"`
}

// DynamicExpert represents a single expert parsed from the generator's JSON output.
type DynamicExpert struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	SystemPrompt string `json:"system_prompt"`
}

// DynamicResponse is the JSON structure returned by the Expert Sourcing Director.
type DynamicResponse struct {
	Experts []DynamicExpert `json:"experts"`
}

// FormatOutput formats an agent's output for inclusion in the Decider's context.
func FormatOutput(agentName, output string) string {
	return fmt.Sprintf("**[%s]**\n%s", agentName, output)
}

// ExtractJSON robustly extracts a JSON object from a string that may contain
// markdown fences or conversational text around it.
func ExtractJSON(raw string, target any) error {
	// Use a simple approach: find first { and last }
	start := -1
	end := -1
	for i, c := range raw {
		if c == '{' && start == -1 {
			start = i
		}
		if c == '}' {
			end = i
		}
	}
	if start == -1 || end == -1 || end <= start {
		return fmt.Errorf("no JSON object found in response")
	}
	return json.Unmarshal([]byte(raw[start:end+1]), target)
}
