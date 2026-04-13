package server

import "html/template"

// --- Template Data Structs ---

// ConfigData is passed to config_badge.html.
type ConfigData struct {
	Provider string
	Model    string
}

// ClarificationData is passed to clarification.html.
type ClarificationData struct {
	Questions        []string
	EncodedContext   string
	UseDynamicAgents bool
}

// AgentPanelData describes a single debater panel in the Wave 2 grid.
type AgentPanelData struct {
	ID    string
	Title string
	Color string
}

// BoardData is passed to board.html.
type BoardData struct {
	SessionID        string
	PulseClass       string
	AgentPanels      []AgentPanelData
	UseDynamicAgents bool
}

// SessionRestoreData is passed to session_restore.html.
type SessionRestoreData struct {
	AgentOutputsJSON template.JS
}

// SSEConnectData is passed to sse_connect.html.
type SSEConnectData struct {
	SessionID string
}

// DynamicPanelData is passed to dynamic_panel.html.
type DynamicPanelData struct {
	Name    string
	CleanID string
}

// buildBoardData constructs the template data for the board layout.
func buildBoardData(sessionID string, isLive bool, useDynamicAgents bool) BoardData {
	pulseClass := ""
	if isLive {
		pulseClass = "animate-pulse streaming-pulse"
	}

	return BoardData{
		SessionID:  sessionID,
		PulseClass: pulseClass,
		AgentPanels: []AgentPanelData{
			{ID: "optimist", Title: "Optimist", Color: "emerald"},
			{ID: "pessimist", Title: "Pessimist", Color: "rose"},
			{ID: "analyst", Title: "Financial Analyst", Color: "amber"},
		},
		UseDynamicAgents: useDynamicAgents,
	}
}

type MemorySnippet struct {
	ID      string
	Content string
}

type MemoriesData struct {
	CoreMemory   map[string]string
	LearnedFacts map[string]string
	Snippets     []MemorySnippet
}
