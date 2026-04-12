package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/golang/glog"
	"github.com/sadlil/boardroom/internal/database"
	"github.com/sadlil/boardroom/internal/llm"
)

// Orchestrator manages the sequential multi-wave debate pipeline.
type Orchestrator struct {
	llmClient llm.Client
	sqlite    *database.SQLiteDB
	memory    *database.VectorMemory
}

// NewOrchestrator creates an Orchestrator backed by the given LLM client.
func NewOrchestrator(client llm.Client, sqlite *database.SQLiteDB, memory *database.VectorMemory) *Orchestrator {
	return &Orchestrator{llmClient: client, sqlite: sqlite, memory: memory}
}

// RunDebate manages the sequential 3-Wave debate.
// Returns an error if a critical failure prevents the debate from completing.
func (o *Orchestrator) RunDebate(ctx context.Context, prompt string, useDynamic bool, callback StreamCallback) error {
	// Build the concurrency semaphore
	sem := o.buildSemaphore()

	// Wave 1: Context Agents
	contextJSON := o.runWave1(ctx, prompt, callback)
	if ctx.Err() != nil {
		return fmt.Errorf("debate cancelled during context analysis: %w", ctx.Err())
	}

	// Wave 2 + Wave 2.5 (parallel)
	wave2Outputs, wave2Err := o.runWave2(ctx, prompt, contextJSON, useDynamic, callback, sem)
	if ctx.Err() != nil {
		return fmt.Errorf("debate cancelled during board debate: %w", ctx.Err())
	}
	if wave2Err != nil && len(wave2Outputs) == 0 {
		return fmt.Errorf("all board debaters failed: %w", wave2Err)
	}

	// Wave 3: The Decider
	wave3Output, wave3Err := o.runWave3(ctx, prompt, contextJSON, wave2Outputs, callback)
	if wave3Err != nil {
		return fmt.Errorf("the Decider agent failed to render a verdict: %w", wave3Err)
	}

	// Background: The Scribe
	go o.runScribe(context.WithoutCancel(ctx), prompt, contextJSON, wave2Outputs, wave3Output)
	return nil
}

// buildSemaphore creates a channel-based semaphore from MAX_CONCURRENT_AGENTS.
func (o *Orchestrator) buildSemaphore() chan struct{} {
	maxConcurrent := 3
	if envMax := os.Getenv("MAX_CONCURRENT_AGENTS"); envMax != "" {
		if parsed, err := strconv.Atoi(envMax); err == nil && parsed > 0 {
			maxConcurrent = parsed
		}
	}
	return make(chan struct{}, maxConcurrent)
}

// runWave1 executes the context agents in parallel and returns the marshaled ContextPayload.
func (o *Orchestrator) runWave1(ctx context.Context, prompt string, callback StreamCallback) string {
	// A) Fetch Static Profile
	glog.Info("[Context] Fetching user profile from SQLite...")
	staticProfile, err := o.sqlite.GetProfile()
	if err != nil {
		glog.Errorf("[Context] ✗ Failed to fetch static profile: %v\n", err)
	}
	if staticProfile == "" {
		staticProfile = "No user profile available. The user has not completed onboarding."
		glog.Info("[Context] No profile found — using default placeholder.")
	} else {
		glog.Infof("[Context] ✓ Profile loaded (%d chars)\n", len(staticProfile))
	}

	// B) Fetch Semantic History (top 3 most relevant past decisions)
	glog.Info("[Context] Querying vector DB for relevant past decisions...")
	historyStr := "No prior decision history available."
	if docs, err := o.memory.Search(ctx, prompt, 3); err == nil && len(docs) > 0 {
		var b strings.Builder
		for i, v := range docs {
			b.WriteString(fmt.Sprintf("\n[Decision %d] %s", i+1, v.Content))
		}
		historyStr = b.String()
		glog.Infof("[Context] ✓ Found %d relevant past decisions\n", len(docs))
	} else if err != nil {
		glog.Errorf("[Context] ✗ Vector search failed: %v\n", err)
	} else {
		glog.Info("[Context] No relevant past decisions found.")
	}

	// C) Inject structured context into the Chief of Staff's system prompt
	contextBlock := fmt.Sprintf(`

══════════════════════════════════════════════
EXECUTIVE MEMORY SYSTEM — CONFIDENTIAL CONTEXT
══════════════════════════════════════════════

## Static User Profile (from onboarding & learned facts)
%s

## Relevant Decision History (semantic matches)
%s

══════════════════════════════════════════════
Use the above context to deeply personalize your analysis.
Reference specific profile details and past decisions when relevant.
══════════════════════════════════════════════`, staticProfile, historyStr)

	augmentedChiefOfStaff := ChiefOfStaff
	augmentedChiefOfStaff.SystemPrompt = ChiefOfStaff.SystemPrompt + contextBlock
	glog.Infof("[Context] Chief of Staff augmented with %d chars of context\n", len(contextBlock))

	var wg sync.WaitGroup
	var mu sync.Mutex
	var userPersonaJSON, macroJSON strings.Builder

	for _, a := range ContextAgents() {
		wg.Add(1)
		go func(agent Agent) {
			defer wg.Done()

			// Use augmented persona if this is the Chief of Staff
			evalAgent := agent
			if agent.ID == ChiefOfStaff.ID {
				evalAgent = augmentedChiefOfStaff
			}

			fullResponse, err := o.executeWithRetry(ctx, evalAgent.ID, evalAgent.Name, evalAgent.SystemPrompt,
				[]llm.Message{{Role: "user", Content: prompt}}, callback)
			if err == nil {
				mu.Lock()
				if evalAgent.ID == ChiefOfStaff.ID {
					userPersonaJSON.WriteString(fullResponse)
				} else {
					macroJSON.WriteString(fullResponse)
				}
				mu.Unlock()
			}
		}(a)
	}

	wg.Wait()
	payload := ContextPayload{
		UserPersona:    userPersonaJSON.String(),
		Macroeconomics: macroJSON.String(),
	}
	payloadBytes, _ := json.MarshalIndent(payload, "", "  ")
	return string(payloadBytes)
}

// runWave2 executes debaters and optionally dynamic experts in parallel.
// Returns the collected outputs and any last error encountered.
func (o *Orchestrator) runWave2(ctx context.Context, prompt, contextJSON string, useDynamic bool, callback StreamCallback, sem chan struct{}) ([]string, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var wave2Outputs []string
	var lastErr error

	// Launch fixed debaters
	for _, a := range DebaterAgents() {
		glog.Infof("Starting debate for agent: %s\n", a.Name)
		wg.Add(1)
		go func(agent Agent) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			enhancedPrompt := "Context Payload:\n" + contextJSON + "\n\nUser Prompt: " + prompt
			fullResponse, err := o.executeWithRetry(ctx, agent.ID, agent.Name, agent.SystemPrompt,
				[]llm.Message{{Role: "user", Content: enhancedPrompt}}, callback)
			mu.Lock()
			if err == nil {
				wave2Outputs = append(wave2Outputs, FormatOutput(agent.Name, fullResponse))
			} else {
				lastErr = fmt.Errorf("agent %s failed: %w", agent.Name, err)
				glog.Errorf("Wave 2 agent %s failed: %v\n", agent.Name, err)
			}
			mu.Unlock()
		}(a)
	}

	// Launch Wave 2.5 in parallel if enabled
	if useDynamic {
		wg.Add(1)
		go func() {
			defer wg.Done()
			o.runDynamicSourcing(ctx, prompt, contextJSON, callback, &wave2Outputs, &mu, sem)
		}()
	}

	wg.Wait()
	return wave2Outputs, lastErr
}

// runWave3 feeds all prior outputs to the Decider for final synthesis.
func (o *Orchestrator) runWave3(ctx context.Context, prompt, contextJSON string, wave2Outputs []string, callback StreamCallback) (string, error) {
	wave3Prompt := "Context Payload:\n" + contextJSON +
		"\n\nDebaters:\n" + strings.Join(wave2Outputs, "\n\n---\n\n") +
		"\n\nUser Procedure Request: " + prompt

	output, err := o.executeWithRetry(ctx, Decider.ID, Decider.Name, Decider.SystemPrompt,
		[]llm.Message{{Role: "user", Content: wave3Prompt}}, callback)
	return output, err
}

// ParseGuests looks for `/invite @RoleName` in the prompt string.
func ParseGuests(prompt string) []string {
	re := regexp.MustCompile(`(?:^|\s)/invite\s+@([A-Za-z0-9_]+)`)
	matches := re.FindAllStringSubmatch(prompt, -1)

	var guests []string
	for _, m := range matches {
		if len(m) > 1 {
			guests = append(guests, m[1])
		}
	}
	return guests
}
