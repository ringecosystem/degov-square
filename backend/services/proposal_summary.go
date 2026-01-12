package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ringecosystem/degov-square/database"
	dbmodels "github.com/ringecosystem/degov-square/database/models"
	"github.com/ringecosystem/degov-square/internal"
	"gorm.io/gorm"
)

// ProposalSummaryService handles AI-generated proposal summaries
type ProposalSummaryService struct {
	db               *gorm.DB
	openRouterClient *internal.OpenRouterClient
}

// NewProposalSummaryService creates a new ProposalSummaryService instance
func NewProposalSummaryService() *ProposalSummaryService {
	return &ProposalSummaryService{
		db:               database.GetDB(),
		openRouterClient: internal.NewOpenRouterClient(),
	}
}

// ProposalSummaryInput represents the input for generating a proposal summary
type ProposalSummaryInput struct {
	ProposalID string `json:"id"`
	ChainID    int    `json:"chain"`
	Indexer    string `json:"indexer"`
}

// GetOrGenerateSummary returns cached summary or generates a new one
func (s *ProposalSummaryService) GetOrGenerateSummary(input ProposalSummaryInput) (string, error) {
	// Check if summary already exists
	var existingSummary dbmodels.ProposalSummary
	err := s.db.Where("proposal_id = ? AND chain_id = ?", input.ProposalID, input.ChainID).First(&existingSummary).Error
	if err == nil {
		slog.Debug("[proposal-summary] Returning cached summary", "proposal_id", input.ProposalID)
		return existingSummary.Summary, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", fmt.Errorf("failed to query existing summary: %w", err)
	}

	// Fetch proposal from indexer
	indexer := internal.NewDegovIndexer(input.Indexer)
	proposal, err := indexer.InspectProposal(input.ProposalID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch proposal: %w", err)
	}

	if proposal == nil {
		return "", fmt.Errorf("proposal with ID %s not found", input.ProposalID)
	}

	// Generate summary using AI
	summary, err := s.generateSummary(proposal.Description)
	if err != nil {
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}

	// Save to database
	id := fmt.Sprintf("%d-%s", input.ChainID, input.ProposalID)
	now := time.Now()
	newSummary := dbmodels.ProposalSummary{
		ID:          id,
		ChainId:     input.ChainID,
		ProposalID:  input.ProposalID,
		Indexer:     &input.Indexer,
		Description: proposal.Description,
		Summary:     summary,
		CTime:       now,
		UTime:       &now,
	}

	if err := s.db.Create(&newSummary).Error; err != nil {
		slog.Warn("[proposal-summary] Failed to save summary to database", "error", err)
		// Still return the generated summary even if saving fails
	}

	return summary, nil
}

// generateSummary uses AI to generate a summary of the proposal description
func (s *ProposalSummaryService) generateSummary(description string) (string, error) {
	systemPrompt := `You are a community governance research consultant providing neutral summary analysis. Summarize articles concisely, based only on their content, avoiding personal opinions. Explain the summary in a way that is easily understandable for community members, regardless of their familiarity with the topic.

Guidelines:

- **Objective**: Present facts only, neutrally.
- **Structured**: Use clear headings and bullet points.
- **Concise**: Maximum 200 words.
- **Accessible**: Use simple language for all community members.

Use this template:

**Summary**

[Concise summary of the article]

**Key Points**

- [Key point 1]
- [Key point 2]
- [Key point 3]`

	userPrompt := fmt.Sprintf(`%s

----
Generate a comprehensive summary of the proposal based on the description provided.`, description)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	response, err := s.openRouterClient.ChatCompletion(ctx, internal.ChatCompletionRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  0.3,
		MaxTokens:    1024,
	})
	if err != nil {
		return "", fmt.Errorf("AI completion failed: %w", err)
	}

	return response, nil
}
