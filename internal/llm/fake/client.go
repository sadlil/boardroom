package fake

import (
	"context"
	"strings"
	"time"

	"github.com/sadlil/boardroom/internal/llm"
)

// Client is a mock LLM client that returns deterministic, role-accurate
// responses for development and testing without hitting a real API.
type Client struct{}

// NewClient creates a new fake LLM client.
func NewClient() *Client {
	return &Client{}
}

// response definitions keyed by detection logic
var agentResponses = map[string]string{

	"clarification": `{"needs_context": true, "questions": ["What is the specific budget range you are working with?", "What is your target timeline for execution?", "Are there any regulatory or compliance constraints to consider?"]}`,

	"chief_of_staff": `**User Profile Assessment**

- **Decision Style**: Data-driven with moderate risk tolerance; prefers structured analysis over gut instinct
- **Core Motivation**: Long-term wealth preservation with selective high-conviction bets
- **Historical Pattern**: Tends to over-analyze and delay action — past decisions show 2-3x longer deliberation than optimal
- **Risk Tolerance**: Moderate (willing to allocate 15-25% of capital to asymmetric opportunities)
- **Key Lesson**: Previous venture in 2024 succeeded specifically because of early commitment; hesitation in 2023 led to missed market timing`,

	"chief_economist": `**Macro Tailwinds**
- Global AI infrastructure spend projected at $420B in 2026 (+35% YoY) — capital is flowing aggressively into adjacent sectors
- Interest rates stabilizing at 3.8% — favorable for leveraged growth plays with 18-24 month horizons
- Talent market correction: senior technical compensation down 12% from 2024 peaks, reducing operational burn rates

**Macro Headwinds**
- Regulatory tightening in EU/APAC markets adding 4-6 month compliance overhead for new market entrants
- Currency volatility (USD/EUR spread at 8.2%) creates FX risk for cross-border revenue models
- Consumer discretionary spending flat — B2C plays face demand compression in H2 2026`,

	"optimist": `**This is a generational opportunity.**

The convergence of **declining talent costs**, **abundant AI infrastructure capital**, and a **regulatory window** that hasn't yet closed creates an asymmetric upside that rarely presents itself. If executed with precision, the potential **3-5x return over 24 months** is well within reach.

**Key Upside Vectors:**
- **First-mover advantage** in an underserved vertical — competitors are 6-12 months behind on technical infrastructure
- **Network effects** compound exponentially once the initial user base crosses the 10K threshold
- **Strategic acquisition target** — three major players are actively looking to acquire capability in this exact space at 8-12x revenue multiples

The user's historical pattern of hesitation is the *only* real risk here. **Move now or lose the window entirely.**`,

	"pessimist": `**This will fail without dramatic intervention.**

The structural weaknesses are being systematically ignored:

- **Cash burn rate** at current projections depletes reserves in 14 months — well before any reasonable path to profitability
- **Single point of failure**: Entire thesis depends on one unproven distribution channel with a 65% historical failure rate for similar ventures
- **Hidden costs**: Regulatory compliance alone will consume 20-30% of operating budget; this is not priced into any current projections
- **Competitive moat is illusory** — the "6-month lead" assumes competitors aren't already building in stealth (they are)

**Worst-Case Scenario**: Full capital loss within 18 months with zero strategic salvage value. The market will not wait for iteration cycles.

The optimist's "generational opportunity" framing is survivorship bias masquerading as analysis.`,

	"analyst": `**Financial Breakdown**

| Metric | Conservative | Base Case | Optimistic |
|--------|-------------|-----------|-----------|
| Initial Capital Required | $180K | $180K | $180K |
| Monthly Burn Rate | $22K | $18K | $15K |
| Runway (months) | 8.2 | 10.0 | 12.0 |
| Time to Revenue | 9 mo | 6 mo | 4 mo |
| Break-even Point | 18 mo | 12 mo | 8 mo |
| 24-month ROI | -35% | +85% | +280% |

**Opportunity Cost Analysis**:
- Alternative deployment in index funds: projected 12-15% annual return (low risk)
- Alternative deployment in T-bills: 4.2% guaranteed return
- **Risk-adjusted expected value** of this venture: +42% (accounting for 40% probability of total loss)

**Recommendation**: The numbers support a **staged deployment** — invest 60% now, hold 40% contingent on month-6 milestone achievement.`,

	"generator": `{
  "experts": [
    {
      "id": "security_expert",
      "name": "Chief Security Officer",
      "system_prompt": "You are the Chief Security Officer. Evaluate all cybersecurity, data privacy, and infrastructure security implications. Focus on attack surfaces, compliance requirements (SOC2, GDPR), and security cost overhead."
    },
    {
      "id": "legal_counsel",
      "name": "General Counsel",
      "system_prompt": "You are the General Counsel. Assess legal liability, intellectual property risks, contractual obligations, and regulatory exposure. Focus on jurisdictional complexity and litigation probability."
    }
  ]
}`,

	"cso": `**Security Assessment**

**Critical**: The proposed architecture has **3 unmitigated attack surfaces** that require immediate attention before launch. SOC2 Type II certification will take 4-6 months and cost approximately $45K — this timeline is non-negotiable for enterprise customers.

Data residency requirements in target markets mandate **region-specific infrastructure**, adding 30% to cloud costs. Without this, the venture faces regulatory shutdown risk in EU markets within 60 days of launch.`,

	"general_counsel": `**Legal Risk Matrix**

**High Risk**: Current IP assignment structure leaves a **critical gap** — all pre-incorporation work product requires retroactive assignment agreements or faces ownership disputes. This is a deal-breaker for any future acquisition or funding round.

**Medium Risk**: The competitive landscape includes 2 active patent holders whose claims overlap with the proposed service model. Freedom-to-operate opinion (~$15K) is recommended before significant capital deployment. Without it, injunction risk is estimated at 15-20%.`,

	"decider": `### The Verdict

**Proceed with a staged deployment strategy** — commit 60% of capital immediately to secure first-mover positioning, with the remaining 40% gated on achieving month-6 revenue milestones.

### The Rationale

The **Chief Financial Officer's staged approach** carried the most weight because it transforms an all-or-nothing bet into a structured experiment with clear exit ramps. The Pessimist's concerns about burn rate and single-channel dependency are valid but are mitigated by the Analyst's contingency structure. The Optimist correctly identifies the closing window — delay is the highest-risk option.

### The Execution Plan

1. **Today**: Execute the initial 60% capital deployment and begin SOC2 certification process (the CSO's timeline is non-negotiable for enterprise readiness)
2. **This Week**: Engage legal counsel for IP assignment cleanup and freedom-to-operate analysis — the General Counsel's flags are blocking risks for any future fundraise
3. **Within 30 Days**: Establish the month-6 milestone criteria (minimum 500 active users, $8K MRR) that will trigger the remaining 40% capital release`,
}

// StreamCompletion simulates an LLM streaming response with role-appropriate content.
func (c *Client) StreamCompletion(ctx context.Context, systemPrompt string, messages []llm.Message, processor llm.StreamProcessor) error {
	response := selectResponse(systemPrompt, messages)
	return streamText(ctx, response, processor)
}

// selectResponse picks the right mock response based on the system prompt and messages.
func selectResponse(systemPrompt string, messages []llm.Message) string {
	payload := strings.ToLower(systemPrompt)
	if len(messages) > 0 {
		payload += " " + strings.ToLower(messages[len(messages)-1].Content)
	}

	switch {
	case strings.Contains(payload, "clarification agent"):
		return agentResponses["clarification"]
	case strings.Contains(payload, "chief of staff"):
		return agentResponses["chief_of_staff"]
	case strings.Contains(payload, "chief economist"):
		return agentResponses["chief_economist"]
	case strings.Contains(payload, "optimist") || strings.Contains(payload, "visionary"):
		return agentResponses["optimist"]
	case strings.Contains(payload, "risk officer") || strings.Contains(payload, "pessimist"):
		return agentResponses["pessimist"]
	case strings.Contains(payload, "financial officer") || strings.Contains(payload, "analyst"):
		return agentResponses["analyst"]
	case strings.Contains(payload, "sourcing director") || strings.Contains(payload, "expert sourcing"):
		return agentResponses["generator"]
	case strings.Contains(payload, "security") || strings.Contains(payload, "cso"):
		return agentResponses["cso"]
	case strings.Contains(payload, "counsel") || strings.Contains(payload, "legal"):
		return agentResponses["general_counsel"]
	case strings.Contains(payload, "decider") || strings.Contains(payload, "final decider"):
		return agentResponses["decider"]
	default:
		return "This is a mock response from the Fake LLM. The pipeline is functioning correctly but no specific agent persona was detected."
	}
}

// streamText simulates token-by-token streaming with artificial delay.
func streamText(ctx context.Context, text string, processor llm.StreamProcessor) error {
	words := strings.Fields(text)
	for i, word := range words {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			chunk := word
			if i < len(words)-1 {
				chunk += " "
			}
			if err := processor(chunk); err != nil {
				return err
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
	return nil
}
