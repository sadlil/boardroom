package agents

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/golang/glog"
	"github.com/sadlil/boardroom/internal/llm"
	"github.com/sadlil/boardroom/ui"
)

// --- Wave 2.5: Dynamic Expert Sourcing ---

// ExpertGenerator is the agent that identifies domain-specific experts needed
// to evaluate the user's dilemma.
var ExpertGenerator = Agent{
	ID:      "generator",
	Name:    "Expert Sourcing Director",
	Persona: "Recruiter",
	Wave:    "dynamic",
	SystemPrompt: `You are the Expert Sourcing Director for an executive Board of Directors. 
Based on the user's dilemma and context, identify exactly 1 to 3 domain-specific experts needed to properly evaluate this situation natively. 
For example, if the dilemma is about hacking, you might summon a "Chief Cybersecurity Officer". If it's about a lawsuit, a "Chief Legal Counsel".

You MUST output ONLY raw, valid JSON. DO NOT use markdown code fences. Your response MUST begin exactly with { and end exactly with }. DO NOT output conversational text before or after the JSON.

Example Output:
{
  "experts": [
    {
      "id": "privacy_officer",
      "name": "Chief Privacy Officer",
      "system_prompt": "You are the Chief Privacy Officer. Evaluate the situation natively."
    }
  ]
}`,
}

const dynamicExpertConstraint = "\n\nCRITICAL CONSTRAINT: You MUST be extremely precious, short, and strictly on point (maximum 2 paragraphs). Provide extremely high value density. Do not use conversational filler."

// runDynamicSourcing executes Wave 2.5: generates dynamic experts and runs them.
// It writes results into wave2Outputs (protected by mu) and uses sem for throttling.
func (o *Orchestrator) runDynamicSourcing(
	ctx context.Context,
	prompt string,
	contextJSON string,
	callback StreamCallback,
	wave2Outputs *[]string,
	mu *sync.Mutex,
	sem chan struct{},
) {
	callback("recruiter", "\n> *Analyzing sourcing requirements against the user's dilemma...*\n")

	// Acquire semaphore for the generator call
	sem <- struct{}{}
	dynJSON, err := o.executeWithRetry(ctx, ExpertGenerator.ID, ExpertGenerator.Name, ExpertGenerator.SystemPrompt,
		[]llm.Message{{Role: "user", Content: "Context Payload:\n" + contextJSON + "\n\nUser Prompt: " + prompt}},
		func(agentID, chunk string) {
			if agentID == "toast" {
				callback("toast", chunk)
			}
		})
	<-sem

	if err != nil {
		return
	}

	var res DynamicResponse
	if err := ExtractJSON(dynJSON, &res); err != nil {
		glog.Errorf("Failed to parse dynamic expert JSON: %v\n", err)
		return
	}

	glog.Infof("Successfully extracted %d dynamic experts from the Generator model.\n", len(res.Experts))
	callback("recruiter", fmt.Sprintf("\n\n**Decision**: Sourced %d specialized experts for immediate evaluation:\n", len(res.Experts)))

	var wgDyn sync.WaitGroup
	idSanitizer := regexp.MustCompile(`[^a-zA-Z0-9]`)

	for i, exp := range res.Experts {
		if i >= 3 {
			break // enforce max 3 dynamic agents
		}

		callback("recruiter", fmt.Sprintf("- **%s**\n", exp.Name))

		cleanID := "dyn_" + idSanitizer.ReplaceAllString(strings.ToLower(exp.ID), "")

		// Render the panel HTML from template
		html, err := ui.RenderToString("dynamic_panel.html", struct {
			Name    string
			CleanID string
		}{Name: exp.Name, CleanID: cleanID})
		if err != nil {
			glog.Errorf("Failed to render dynamic panel template: %v\n", err)
			continue
		}

		callback("dynamic_panel", html)

		expertPrompt := exp.SystemPrompt + dynamicExpertConstraint

		// Spin up parallel dynamic experts with semaphore throttling
		wgDyn.Add(1)
		go func(name, id, sysPrompt string) {
			defer wgDyn.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			glog.Infof("Starting dynamic debate for agent: %s (%s)\n", name, id)

			out, _ := o.executeWithRetry(ctx, id, name, sysPrompt, []llm.Message{{Role: "user", Content: "Context Payload:\n" + contextJSON + "\n\nUser Prompt: " + prompt}}, callback)
			if out != "" {
				mu.Lock()
				*wave2Outputs = append(*wave2Outputs, FormatOutput(name, out))
				mu.Unlock()
			}
		}(exp.Name, cleanID, expertPrompt)
	}

	wgDyn.Wait()
}
