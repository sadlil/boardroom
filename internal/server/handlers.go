package server

import (
	"encoding/base64"
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/sadlil/boardroom/internal/agents"
	"github.com/sadlil/boardroom/ui"
)

// handleInit checks if a user profile exists and serves either the Command Center or Onboarding form.
func (h *Handler) handleInit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	profile, err := h.sqlite.GetProfile()
	if err != nil {
		glog.Errorf("Error fetching profile: %v", err)
	}

	if profile == "" {
		ui.Render(w, "onboarding.html", nil)
	} else {
		ui.Render(w, "command_center.html", nil)
	}
}

// handleOnboard saves the initial user profile and swaps the UI to the Command Center.
func (h *Handler) handleOnboard(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Build structured profile from form fields
	profile := map[string]string{
		"role":             r.FormValue("role"),
		"industry":         r.FormValue("industry"),
		"experience_level": r.FormValue("experience_level"),
		"risk_tolerance":   r.FormValue("risk_tolerance"),
		"goals":            r.FormValue("goals"),
		"constraints":      r.FormValue("constraints"),
		"background":       r.FormValue("background"),
	}

	profileJSON, err := json.Marshal(profile)
	if err != nil {
		glog.Errorf("Failed to marshal profile: %v", err)
		http.Error(w, "Failed to process profile", http.StatusInternalServerError)
		return
	}

	if err := h.sqlite.SaveProfile(string(profileJSON)); err != nil {
		glog.Errorf("Failed to save profile: %v", err)
		http.Error(w, "Failed to save profile", http.StatusInternalServerError)
		return
	}

	glog.Infof("User profile saved: role=%s, industry=%s, experience=%s, risk=%s",
		profile["role"], profile["industry"], profile["experience_level"], profile["risk_tolerance"])

	w.Header().Set("Content-Type", "text/html")
	ui.Render(w, "command_center.html", nil)
}

// handleConfig returns the current LLM provider/model badge.
func (h *Handler) handleConfig(w http.ResponseWriter, r *http.Request) {
	provider := os.Getenv("LLM_PROVIDER")
	if provider == "" {
		provider = "ollama"
	}
	model := os.Getenv("LLM_MODEL")
	if model == "" && provider == "ollama" {
		model = "gemma3:1b"
	}

	w.Header().Set("Content-Type", "text/html")
	ui.Render(w, "config_badge.html", ConfigData{Provider: provider, Model: model})
}

// handleGetSession restores a completed session from memory.
func (h *Handler) handleGetSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("id")
	session := h.sessions.Get(sessionID)
	if session == nil {
		http.Error(w, "Session not found or expired", http.StatusNotFound)
		return
	}

	memJSON, _ := json.Marshal(session.AgentOutputs)
	boardHTML, _ := ui.RenderToString("board.html", buildBoardData(false, session.UseDynamicAgents))

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(boardHTML))
	ui.Render(w, "session_restore.html", SessionRestoreData{
		AgentOutputsJSON: template.JS(memJSON),
	})
}

// handleStartDebate processes the debate form submission.
func (h *Handler) handleStartDebate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	prompt := r.FormValue("prompt")
	previousContext := r.FormValue("previous_context")
	action := r.FormValue("action")
	useDynamicAgents := r.FormValue("dynamic_agents") == "true"

	glog.Infof("Dynamic agents enabled: %v", useDynamicAgents)

	if prompt == "" && action != "skip" {
		http.Error(w, "Prompt is required", http.StatusBadRequest)
		return
	}

	fullPrompt := buildFullPrompt(prompt, previousContext, action)

	// Wave 0: Clarification check (only on first submission)
	if previousContext == "" {
		clarification := h.checkClarification(r, fullPrompt)
		if clarification.NeedsContext && len(clarification.Questions) > 0 {
			w.Header().Set("Content-Type", "text/html")
			ui.Render(w, "clarification.html", ClarificationData{
				Questions:        clarification.Questions,
				EncodedContext:   base64.StdEncoding.EncodeToString([]byte(fullPrompt)),
				UseDynamicAgents: useDynamicAgents,
			})
			return
		}
	}

	// Create session and render the board
	sessionID := h.sessions.Create(fullPrompt, useDynamicAgents)

	w.Header().Set("HX-Push-Url", "?session="+sessionID)
	w.Header().Set("Content-Type", "text/html")

	boardHTML, _ := ui.RenderToString("board.html", buildBoardData(true, useDynamicAgents))
	w.Write([]byte(boardHTML))
	ui.Render(w, "sse_connect.html", SSEConnectData{SessionID: sessionID})
}

// --- Private helpers ---

// buildFullPrompt constructs the final prompt from user input and any prior context.
func buildFullPrompt(prompt, previousContext, action string) string {
	if previousContext == "" {
		return prompt
	}

	decodedContext, err := base64.StdEncoding.DecodeString(previousContext)
	if err == nil {
		previousContext = string(decodedContext)
	}

	if action == "skip" {
		return previousContext + "\n\nUser skipped the clarification questionnaire."
	}
	return previousContext + "\n\nUser added clarification: " + prompt
}

// checkClarification runs Wave 0 and returns the clarification result.
func (h *Handler) checkClarification(r *http.Request, prompt string) *agents.ClarificationResult {
	glog.Info("Evaluating prompt for clarification...")
	result, err := h.orchestrator.CheckClarification(r.Context(), prompt)
	if err != nil {
		glog.Errorf("Clarification failed: %v", err)
		return &agents.ClarificationResult{NeedsContext: false}
	}
	return result
}

// handleGetMemories fetches all vector memories and the user profile for display.
func (h *Handler) handleGetMemories(w http.ResponseWriter, r *http.Request) {
	profile, _ := h.sqlite.GetProfile()

	docs, err := h.memory.GetAllDocuments(r.Context())
	if err != nil {
		glog.Errorf("Failed to retrieve vector memories: %v", err)
		http.Error(w, "Failed to retrieve memories", http.StatusInternalServerError)
		return
	}

	snippets := make([]MemorySnippet, 0, len(docs))
	for _, doc := range docs {
		snippets = append(snippets, MemorySnippet{
			ID:      doc.ID,
			Content: doc.Content,
		})
	}

	var coreMemory map[string]string

	learnedFacts, err := h.sqlite.GetUserFacts()
	if err != nil {
		glog.Errorf("Failed to retrieve user facts: %v", err)
	}

	if profile != "" {
		if err := json.Unmarshal([]byte(profile), &coreMemory); err != nil {
			glog.Errorf("Failed to parse core memory JSON: %v | Raw: %s", err, profile)
			coreMemory = map[string]string{"Raw Profile": profile}
		}
	}

	data := MemoriesData{
		CoreMemory:   coreMemory,
		LearnedFacts: learnedFacts,
		Snippets:     snippets,
	}

	w.Header().Set("Content-Type", "text/html")
	ui.Render(w, "memories_modal.html", data)
}
