---
name: Boardroom Architecture and Rules
description: Context on the internal Go architecture, backend rendering structure, and multi-agent system rules.
---

# Boardroom Internal Agent Knowledge

If you are an AI agent analyzing or modifying this codebase, refer to this document for strict architectural guidelines.

## 1. System Architecture

- **Entrypoint**: `cmd/boardroom/main.go`
- **Frontend / UI Rendering**:
  - We do **NOT** use heavy frontend frameworks (React/Vue/Tailwind Build). 
  - We use standard Go `html/template` for server-side HTML rendering. Templates are located in `ui/templates/` and compiled/served via `ui/embed.go`.
  - Dynamic UI updates are driven by a native Javascript `EventSource` (SSE) combined with `marked.js` for client-side markdown parsing. We no longer use HTMX's SSE extension.
- **Backend / Server**: Modularized inside `internal/server/`.
  - `handlers.go` contains pure HTTP logic.
  - `routes.go` wires the multiplexer.
  - `templates.go` defines all strongly-typed structs passed to the UI.
  - `session.go` provides a thread-safe LRU cache (using `hashicorp/golang-lru/v2`) protecting active debates from memory leaks.
- **LLM Interface**: We abstract interfaces through `internal/llm/llm.go` to allow seamless provider hot-swapping (`ollama`, `gemini`, or `fake` for testing).

## 2. Multi-Agent Orchestration

The orchestration logic lives in `internal/agents/`, which is strictly modularized:
- **`orchestrator.go`** handles pure wave sequencing (`RunDebate`).
- **`executor.go`** handles LLM call retries and toast notification generation.
- **Agent Definitions**: Each agent identity (ID, Name, Persona, Prompt) is isolated in its respective role file: `clarifier.go`, `context_agents.go`, `debater_agents.go`, `decider.go`, `dynamic.go`. 

The agents execute in parallel within isolated **Waves**:
- **Wave 0** (Clarification) -> **Wave 1** (Context) -> **Wave 2** (Debaters) + **Wave 2.5** (Dynamic Experts) -> **Wave 3** (Decider).
- **CRITICAL**: The Decider (Wave 3) cannot run until Wave 2 and 2.5 successfully compile their inputs into an array context sheet.
- **Constraints**: Agents are restricted from asking follow-up questions to the user during the debate. Their prompts use strong constraints (`CRITICAL CONSTRAINT: You MUST be extremely precious...`).

## 3. UI State and Dynamic Behaviors

When modifying the UI or backend handlers, keep these unique behaviors in mind to prevent catastrophic DOM bugs:
- **URL Pushing**: `handleStartDebate` uses `HX-Push-Url` to force the browser to `/?session={sessionID}`.
- **Session Restoration**: On page reload, `handleGetSession` fetches the memory buffer (`SessionData.AgentOutputs`), re-renders the base HTML via templates, and manually injects the markdown back into the browser using the `session_restore.html` template snippet.
- **Native SSE Locks**: In `sse.go`, we explicitly mark `SessionData.Status = "completed"` after the debate concludes. The native JS `EventSource` checks this and cleanly disconnects to prevent infinite reconnections.
- **Dynamic Agent Registration**: Dynamic agents (Wave 2.5) are injected into the DOM as pure HTML strings. The Javascript explicitly calls `window.__registerDynAgent` the moment they enter the DOM to capture their SSE stream without missing tokens.
- **DOM Folding (UX)**: Wave 1 and Wave 2 rely on `<details>` containers for aesthetics. Javascript utilizes structural latches (`window.hasFoldedWave1`) to collapse previous folders precisely *once* when a new wave begins streaming. Do not modify these boolean latches.
