package agents

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
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

func ExtractJSON(raw string, target any) error {
	// 1. Try to find markdown code block
	re := regexp.MustCompile("(?s)```(?:json)?\n(.*?)\n```")
	matches := re.FindStringSubmatch(raw)
	if len(matches) > 1 {
		if err := json.Unmarshal([]byte(matches[1]), target); err == nil {
			return nil
		}
	}

	// 2. Fallback to extracting everything between first { and last }
	start := strings.IndexByte(raw, '{')
	end := strings.LastIndexByte(raw, '}')
	if start != -1 && end != -1 && end > start {
		return json.Unmarshal([]byte(raw[start:end+1]), target)
	}

	return fmt.Errorf("no JSON object found in response")
}
