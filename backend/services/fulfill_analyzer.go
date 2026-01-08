package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"text/template"
	"time"

	"github.com/ringecosystem/degov-square/internal"
	"github.com/ringecosystem/degov-square/internal/templates"
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

// FulfillAnalyzer handles AI-based vote analysis for proposal fulfillment
type FulfillAnalyzer struct {
	client       *internal.OpenRouterClient
	systemPrompt string
	userTemplate *template.Template
}

// NewFulfillAnalyzer creates a new FulfillAnalyzer
func NewFulfillAnalyzer(client *internal.OpenRouterClient) (*FulfillAnalyzer, error) {
	// Load system prompt from template
	systemPromptBytes, err := templates.TemplateFS.ReadFile("prompts/fulfill-vote-analysis.system.md")
	if err != nil {
		return nil, fmt.Errorf("failed to load system prompt template: %w", err)
	}

	// Load user prompt template
	userPromptBytes, err := templates.TemplateFS.ReadFile("prompts/fulfill-vote-analysis.user.md")
	if err != nil {
		return nil, fmt.Errorf("failed to load user prompt template: %w", err)
	}

	userTemplate, err := template.New("user-prompt").Parse(string(userPromptBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to parse user prompt template: %w", err)
	}

	return &FulfillAnalyzer{
		client:       client,
		systemPrompt: string(systemPromptBytes),
		userTemplate: userTemplate,
	}, nil
}

// AnalyzeVotes analyzes votes using AI and returns the analysis result
func (a *FulfillAnalyzer) AnalyzeVotes(ctx context.Context, voteCasts []VoteCastInfo) (*AnalysisResult, error) {
	userPrompt, err := a.buildUserPrompt(voteCasts)
	if err != nil {
		return nil, fmt.Errorf("failed to build user prompt: %w", err)
	}

	content, err := a.client.ChatCompletion(ctx, internal.ChatCompletionRequest{
		SystemPrompt: a.systemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  0.3,
		MaxTokens:    4096,
	})
	if err != nil {
		return nil, err
	}

	// Parse the JSON response from content
	result, err := parseAnalysisResult(content)
	if err != nil {
		slog.Warn("Failed to parse AI response", "content", content, "error", err)
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return result, nil
}

// buildUserPrompt builds the user prompt from template and vote data
func (a *FulfillAnalyzer) buildUserPrompt(voteCasts []VoteCastInfo) (string, error) {
	votesJSON, err := json.MarshalIndent(voteCasts, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal votes: %w", err)
	}

	data := map[string]interface{}{
		"VotesJSON": string(votesJSON),
	}

	var buf bytes.Buffer
	if err := a.userTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
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
