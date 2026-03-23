package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sadlil/boardroom/internal/llm"
)

// ScribeAgent defines the persona for the memory manager.
var ScribeAgent = Agent{
	ID:   "scribe",
	Name: "The Scribe",
	SystemPrompt: `You are The Scribe, an autonomous background data processor for a decision-intelligence platform.
Your job is to read a FULL debate transcript (User Dilemma, Context Analysis, Board Debate, and Final Verdict) and output a strict JSON payload.

## Your Two Responsibilities

### 1. Extract New Static User Facts
Analyze the user's provided context text only to identify NEW, persistent facts about the user that should be saved to their static profile. These facts should be structured and categorized for optimal AI consumption.

Categories of facts to look for:
- **Identity**: job title changes, company name, team size, location
- **Financial**: budget ranges, revenue figures, funding status, salary expectations
- **Goals**: stated objectives, timelines, milestones
- **Constraints**: limitations like budget caps, regulatory requirements, time pressure, personal obligations
- **Preferences**: decision-making style, communication preferences, priorities
- **Domain**: technical skills, industry expertise, certifications

Rules:
- Only extract facts that are EXPLICITLY stated or strongly implied by the user's own words.
- Do NOT infer personality traits or make subjective judgments.
- If an existing fact conflicts with a new one, note the updated value.
- If no genuinely new facts exist, set the field to an empty string "".

### 2. Create a Semantic Decision Summary
Write a dense 2-4 sentence summary capturing:
- The specific dilemma the user faced
- The core arguments for and against
- The final verdict (Yes/No/Pivot) and the decisive reasoning
- Any quantitative data points mentioned

This summary will be embedded as a vector for future semantic retrieval, so optimize for search relevance.

## Output Format
Respond ONLY with valid JSON. No markdown formatting, no code blocks, no explanations.
{
  "new_facts": {
    "identity": "Any new identity facts discovered (or empty string)",
    "financial": "Any new financial facts (or empty string)",
    "goals": "Any new goals mentioned (or empty string)",
    "constraints": "Any new constraints identified (or empty string)",
    "preferences": "Any new preferences observed (or empty string)",
    "domain": "Any new domain expertise revealed (or empty string)"
  },
  "decision_summary": "Dense semantic summary of the dilemma, arguments, verdict, and key data points."
}
`,
}

// ScribeOutput represents the structured extraction from the Scribe agent.
type ScribeOutput struct {
	NewFacts        ScribeFacts `json:"new_facts"`
	DecisionSummary string      `json:"decision_summary"`
}

// ScribeFacts represents categorized user facts extracted by the Scribe.
type ScribeFacts struct {
	Identity    string `json:"identity"`
	Financial   string `json:"financial"`
	Goals       string `json:"goals"`
	Constraints string `json:"constraints"`
	Preferences string `json:"preferences"`
	Domain      string `json:"domain"`
}

// HasContent returns true if any fact category contains meaningful content.
func (f ScribeFacts) HasContent() bool {
	return f.Identity != "" || f.Financial != "" || f.Goals != "" ||
		f.Constraints != "" || f.Preferences != "" || f.Domain != ""
}

// FormatForProfile serializes non-empty facts into a structured append string.
func (f ScribeFacts) FormatForProfile() string {
	var parts []string
	if f.Identity != "" {
		parts = append(parts, fmt.Sprintf("  \"identity\": %q", f.Identity))
	}
	if f.Financial != "" {
		parts = append(parts, fmt.Sprintf("  \"financial\": %q", f.Financial))
	}
	if f.Goals != "" {
		parts = append(parts, fmt.Sprintf("  \"goals\": %q", f.Goals))
	}
	if f.Constraints != "" {
		parts = append(parts, fmt.Sprintf("  \"constraints\": %q", f.Constraints))
	}
	if f.Preferences != "" {
		parts = append(parts, fmt.Sprintf("  \"preferences\": %q", f.Preferences))
	}
	if f.Domain != "" {
		parts = append(parts, fmt.Sprintf("  \"domain\": %q", f.Domain))
	}
	return "{\n" + strings.Join(parts, ",\n") + "\n}"
}

// runScribe processes the debate and updates SQLite/VectorDB memories silently in the background.
func (o *Orchestrator) runScribe(ctx context.Context, prompt, contextJSON string, wave2 []string, deciderOutput string) {
	startTime := time.Now()
	log.Println("[Scribe] Starting background memory management")
	log.Printf("[Scribe] Dilemma length: %d chars | Context: %d chars | Debaters: %d | Verdict: %d chars",
		len(prompt), len(contextJSON), len(wave2), len(deciderOutput))

	// Build the full transcript for analysis
	transcript := fmt.Sprintf(
		"=== USER DILEMMA ===\n%s\n\n=== CONTEXT ANALYSIS ===\n%s\n\n=== BOARD DEBATE ===\n%s\n\n=== FINAL VERDICT ===\n%s",
		prompt, contextJSON, strings.Join(wave2, "\n\n---\n\n"), deciderOutput,
	)
	log.Printf("[Scribe] Transcript compiled: %d total chars", len(transcript))

	// Execute the Scribe LLM
	log.Println("[Scribe] Invoking LLM for fact extraction and summary generation...")
	output, err := o.executeWithRetry(ctx, ScribeAgent.ID, ScribeAgent.Name, ScribeAgent.SystemPrompt,
		[]llm.Message{{Role: "user", Content: transcript}}, func(agentID, chunk string) {})
	if err != nil {
		log.Printf("[Scribe] ✗ LLM invocation failed after %.2fs: %v", time.Since(startTime).Seconds(), err)
		return
	}
	log.Printf("[Scribe] LLM responded: %d chars in %.2fs", len(output), time.Since(startTime).Seconds())

	// Clean output (strip markdown code blocks if present)
	cleaned := strings.TrimSpace(output)
	if strings.HasPrefix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```json")
		cleaned = strings.TrimPrefix(cleaned, "```")
		cleaned = strings.TrimSuffix(cleaned, "```")
		cleaned = strings.TrimSpace(cleaned)
	}

	// Parse JSON output
	var result ScribeOutput
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		log.Printf("[Scribe] JSON parse failed: %v", err)
		log.Printf("[Scribe] Raw output (first 500 chars): %.500s", cleaned)
		return
	}
	log.Println("[Scribe] JSON parsed successfully")

	// ── 1. Update SQLite Profile ──
	if result.NewFacts.HasContent() {
		log.Println("[Scribe] New facts detected:")
		if result.NewFacts.Identity != "" {
			log.Printf("[Scribe]   → Identity: %s", result.NewFacts.Identity)
		}
		if result.NewFacts.Financial != "" {
			log.Printf("[Scribe]   → Financial: %s", result.NewFacts.Financial)
		}
		if result.NewFacts.Goals != "" {
			log.Printf("[Scribe]   → Goals: %s", result.NewFacts.Goals)
		}
		if result.NewFacts.Constraints != "" {
			log.Printf("[Scribe]   → Constraints: %s", result.NewFacts.Constraints)
		}
		if result.NewFacts.Preferences != "" {
			log.Printf("[Scribe]   → Preferences: %s", result.NewFacts.Preferences)
		}
		if result.NewFacts.Domain != "" {
			log.Printf("[Scribe]   → Domain: %s", result.NewFacts.Domain)
		}

		currentProfile, _ := o.sqlite.GetProfile()
		appendBlock := "\n\n--- Learned Facts (auto-updated) ---\n" + result.NewFacts.FormatForProfile()

		var newProfile string
		if currentProfile != "" {
			newProfile = currentProfile + appendBlock
		} else {
			newProfile = appendBlock
		}

		if err := o.sqlite.SaveProfile(newProfile); err != nil {
			log.Printf("[Scribe] SQLite profile update failed: %v", err)
		} else {
			log.Printf("[Scribe] SQLite profile updated (%d → %d chars)", len(currentProfile), len(newProfile))
		}
	} else {
		log.Println("[Scribe] No new static facts discovered in this debate.")
	}

	// ── 2. Insert into Vector Database ──
	if result.DecisionSummary != "" {
		docID := uuid.New().String()
		metadata := map[string]string{
			"type":       "decision_summary",
			"dilemma":    prompt,
			"created_at": time.Now().UTC().Format(time.RFC3339),
		}
		log.Printf("[Scribe] Inserting vector document: id=%s, summary_len=%d", docID, len(result.DecisionSummary))
		log.Printf("[Scribe] Summary preview: %.200s", result.DecisionSummary)

		if err := o.memory.AddDocument(ctx, docID, result.DecisionSummary, metadata); err != nil {
			log.Printf("[Scribe] Vector DB insert failed: %v", err)
		} else {
			log.Printf("[Scribe] Vector memory updated (doc_id=%s)", docID)
		}
	} else {
		log.Println("[Scribe] No decision summary generated — skipping vector insert.")
	}

	log.Printf("[Scribe] Background memory management completed in %.2fs", time.Since(startTime).Seconds())
}
