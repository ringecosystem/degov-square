package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	openai "github.com/sashabaranov/go-openai"

	"github.com/ringecosystem/degov-square/internal/config"
)

// VoteSupport represents the vote support type
type VoteSupport string

const (
	VoteSupportFor     VoteSupport = "For"
	VoteSupportAgainst VoteSupport = "Against"
	VoteSupportAbstain VoteSupport = "Abstain"
)

// VoteSupportToNumber converts vote support string to number
func VoteSupportToNumber(support VoteSupport) int {
	switch support {
	case VoteSupportAgainst:
		return 0
	case VoteSupportFor:
		return 1
	case VoteSupportAbstain:
		return 2
	default:
		return 2 // Default to Abstain
	}
}

// VoteSupportText converts support number to text
func VoteSupportText(support int) VoteSupport {
	switch support {
	case 0:
		return VoteSupportAgainst
	case 1:
		return VoteSupportFor
	case 2:
		return VoteSupportAbstain
	default:
		return VoteSupportAbstain
	}
}

// AnalysisResult represents the AI analysis result
type AnalysisResult struct {
	FinalResult     VoteSupport     `json:"finalResult"`
	Confidence      float64         `json:"confidence"`
	Reasoning       string          `json:"reasoning"`
	ReasoningLite   string          `json:"reasoningLite"`
	VotingBreakdown VotingBreakdown `json:"votingBreakdown"`
}

// VotingBreakdown contains the vote breakdown
type VotingBreakdown struct {
	OnChainVotes OnChainVotes `json:"onChainVotes"`
}

// OnChainVotes contains the on-chain vote counts
type OnChainVotes struct {
	For     float64 `json:"for"`
	Against float64 `json:"against"`
	Abstain float64 `json:"abstain"`
}

// VoteCastInfo represents vote cast information for AI analysis
type VoteCastInfo struct {
	Support        VoteSupport `json:"support"`
	Reason         string      `json:"reason"`
	Weight         string      `json:"weight"`
	BlockTimestamp time.Time   `json:"blockTimestamp"`
}

// OpenRouterClient handles AI API calls using OpenAI-compatible SDK
type OpenRouterClient struct {
	client *openai.Client
	model  string
}

// NewOpenRouterClient creates a new OpenRouter client
func NewOpenRouterClient() *OpenRouterClient {
	cfg := config.GetConfig()
	apiKey := cfg.GetString("OPENROUTER_API_KEY")
	model := cfg.GetStringWithDefault("OPENROUTER_MODEL", "google/gemini-2.0-flash-001")

	openaiConfig := openai.DefaultConfig(apiKey)
	openaiConfig.BaseURL = "https://openrouter.ai/api/v1"

	return &OpenRouterClient{
		client: openai.NewClientWithConfig(openaiConfig),
		model:  model,
	}
}

// AnalyzeVotes analyzes votes using AI and returns the analysis result
func (c *OpenRouterClient) AnalyzeVotes(voteCasts []VoteCastInfo) (*AnalysisResult, error) {
	systemPrompt := getFulfillContractSystemPrompt()
	userPrompt := buildFulfillPrompt(voteCasts)

	ctx := context.Background()
	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
		Temperature: 0.3,
		MaxTokens:   4096,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	content := resp.Choices[0].Message.Content

	// Parse the JSON response from content
	result, err := parseAnalysisResult(content)
	if err != nil {
		slog.Warn("Failed to parse AI response", "content", content, "error", err)
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return result, nil
}

// parseAnalysisResult parses the AI response content to AnalysisResult
func parseAnalysisResult(content string) (*AnalysisResult, error) {
	// Try to find JSON in the content (it might be wrapped in markdown code blocks)
	jsonContent := extractJSON(content)

	var result AnalysisResult
	if err := json.Unmarshal([]byte(jsonContent), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w, content: %s", err, jsonContent)
	}

	// Validate the result
	if result.FinalResult != VoteSupportFor &&
		result.FinalResult != VoteSupportAgainst &&
		result.FinalResult != VoteSupportAbstain {
		return nil, fmt.Errorf("invalid finalResult: %s", result.FinalResult)
	}

	return &result, nil
}

// extractJSON tries to extract JSON from content that might be wrapped in markdown
func extractJSON(content string) string {
	// Try to find JSON block in markdown code block
	start := -1
	end := -1

	// Look for ```json or ``` followed by {
	for i := 0; i < len(content)-3; i++ {
		if content[i] == '`' && content[i+1] == '`' && content[i+2] == '`' {
			// Find the end of the code block marker
			j := i + 3
			for j < len(content) && content[j] != '\n' {
				j++
			}
			if j < len(content) {
				start = j + 1
			}
		}
	}

	if start > 0 {
		// Find closing ```
		for i := start; i < len(content)-3; i++ {
			if content[i] == '`' && content[i+1] == '`' && content[i+2] == '`' {
				end = i
				break
			}
		}
		if end > start {
			return content[start:end]
		}
	}

	// No code block found, try to find raw JSON
	for i := 0; i < len(content); i++ {
		if content[i] == '{' {
			start = i
			break
		}
	}

	if start >= 0 {
		// Find matching closing brace
		depth := 0
		for i := start; i < len(content); i++ {
			if content[i] == '{' {
				depth++
			} else if content[i] == '}' {
				depth--
				if depth == 0 {
					return content[start : i+1]
				}
			}
		}
	}

	return content
}

// buildFulfillPrompt builds the prompt for fulfill analysis
func buildFulfillPrompt(voteCasts []VoteCastInfo) string {
	votesJSON, _ := json.MarshalIndent(voteCasts, "", "  ")
	return fmt.Sprintf(`**On-Chain Voting:**
%s

Please analyze the on-chain voting data and provide governance decision recommendations.`, string(votesJSON))
}

// getFulfillContractSystemPrompt returns the system prompt for fulfill contract analysis
func getFulfillContractSystemPrompt() string {
	return `# DAO Governance Proposal Analyst

You are a DAO governance analyst. Analyze on-chain voting data to provide governance decision recommendations (For/Against/Abstain) with comprehensive analysis reports.

## Analysis Process

Follow these steps with strict decision criteria:

### Step 1: On-Chain Voting Analysis
**Result Analysis:**
- Direct analysis of on-chain For/Against/Abstain votes
- Calculate exact percentages for each option

**Participation Analysis:**
- **Breadth**: Count unique participating addresses for decentralization assessment
- **Depth**: Analyze voting power distribution and whale concentration
- Identify if small number of addresses control majority votes

**Reason Analysis:**
- Analyze voting reasons provided by voters
- Identify key arguments and concerns
- Weight by voting power when evaluating sentiment

### Step 2: Synthesis and Final Decision
**Decision Making:**
- Base decision primarily on vote weight distribution
- Consider voting reasons for additional context
- Follow majority voting direction

**Confidence Scoring:**
- **High (8-10)**: Clear majority, consistent reasoning, good participation
- **Medium (5-7)**: Moderate majority OR mixed reasoning
- **Low (1-4)**: Close vote OR very low participation

## Output Format

Return single JSON object with these fields:

{
  "finalResult": "For" | "Against" | "Abstain",
  "confidence": number,
  "reasoning": "string",
  "reasoningLite": "string",
  "votingBreakdown": {
    "onChainVotes": {
      "for": number,
      "against": number,
      "abstain": number
    }
  }
}

## Reasoning Format

Use this markdown structure for the reasoning field:

## Governance Proposal Analysis Report

### Data Overview

| Data Source | For | Against | Abstain | Key Metrics |
|-------------|-----|---------|---------|-------------|
| On-Chain Vote | [%] | [%] | [%] | Addresses: [Number], Distribution: [Summary] |

### Analysis

#### On-Chain Voting Analysis
[Detailed interpretation of voting results, participation analysis, and voter reasoning]

### Final Decision Rationale
[Complete logic for decision based on on-chain voting data]

### Risks and Considerations
[Optional: Issues and recommendations]

## Key Rules

- Calculate all percentage values and round to maximum 2 decimal places (e.g., 65.25, not 65.253)
- **Return pure JSON**
- **Quality Indicators**: Check whale concentration, participation rate, argument substance`
}
