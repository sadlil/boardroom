package agents

import (
	"context"

	"github.com/sadlil/boardroom/internal/llm"
)

// Clarifier is the Wave 0 gatekeeper agent.
var Clarifier = Agent{
	ID:      "clarification",
	Name:    "Clarification Agent",
	Persona: "Gatekeeper",
	Wave:    "clarification",
	SystemPrompt: `You are the Clarification Agent. Your job is to act as a gatekeeper for an executive Board of Directors.
Analyze the user's dilemma. 
- If the dilemma provides enough concrete details for the board to debate effectively, respond strictly with: {"needs_context": false, "questions": []}
- If it is too vague (e.g., "Should I quit my job?"), respond strictly with: {"needs_context": true, "questions": ["Question 1?", "Question 2?"]}

CRITICAL CONSTRAINTS: 
1. You must output ONLY raw, valid JSON. 
2. DO NOT wrap the JSON in markdown code blocks (no '` + "```json" + `'). 
3. Limit to a maximum of 3 highly specific, short questions.`,
}

// CheckClarification (Wave 0) evaluates whether the prompt has enough context
// for the board to debate effectively.
func (o *Orchestrator) CheckClarification(ctx context.Context, prompt string) (*ClarificationResult, error) {
	fullResponse, err := o.executeWithRetry(ctx, Clarifier.ID, Clarifier.Name, Clarifier.SystemPrompt,
		[]llm.Message{{Role: "user", Content: prompt}},
		func(agentID, chunk string) {})
	if err != nil {
		return nil, err
	}

	var res ClarificationResult
	if err := ExtractJSON(fullResponse, &res); err != nil {
		return &ClarificationResult{NeedsContext: false}, nil
	}
	return &res, nil
}
