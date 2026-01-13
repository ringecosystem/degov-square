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
	"github.com/ringecosystem/degov-square/internal/utils"
	"gorm.io/gorm"
)

// ProposalSummaryService handles AI-generated proposal summaries
type ProposalSummaryService struct {
	db               *gorm.DB
	openRouterClient *internal.OpenRouterClient
	daoConfigService *DaoConfigService
}

// NewProposalSummaryService creates a new ProposalSummaryService instance
func NewProposalSummaryService() *ProposalSummaryService {
	return &ProposalSummaryService{
		db:               database.GetDB(),
		openRouterClient: internal.NewOpenRouterClient(),
		daoConfigService: NewDaoConfigService(),
	}
}

// ProposalSummaryInput represents the input for generating a proposal summary
type ProposalSummaryInput struct {
	ProposalID string `json:"id"`
	DaoCode    string `json:"daoCode"`
}

// GetOrGenerateSummary returns cached summary or generates a new one
func (s *ProposalSummaryService) GetOrGenerateSummary(input ProposalSummaryInput) (string, error) {
	// Get DAO config to obtain indexer endpoint and chain ID
	daoConfig, err := s.daoConfigService.StandardConfig(input.DaoCode)
	if err != nil {
		return "", fmt.Errorf("failed to get dao config for %s: %w", input.DaoCode, err)
	}

	chainID := daoConfig.Chain.ID
	indexerEndpoint := daoConfig.Indexer.Endpoint

	slog.Info("[proposal-summary] Looking for cached summary", "proposal_id", input.ProposalID, "chain_id", chainID, "dao_code", input.DaoCode)

	// Check if summary already exists
	var existingSummary dbmodels.ProposalSummary
	err = s.db.Where("proposal_id = ? AND chain_id = ?", input.ProposalID, chainID).First(&existingSummary).Error
	if err == nil {
		slog.Info("[proposal-summary] Returning cached summary", "proposal_id", input.ProposalID, "id", existingSummary.ID)
		return existingSummary.Summary, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		slog.Error("[proposal-summary] Database query error", "error", err)
		return "", fmt.Errorf("failed to query existing summary: %w", err)
	}

	slog.Info("[proposal-summary] No cached summary found, generating new one", "proposal_id", input.ProposalID)

	// Fetch proposal from indexer
	indexer := internal.NewDegovIndexer(indexerEndpoint)
	proposal, err := indexer.InspectProposal(input.ProposalID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch proposal: %w", err)
	}

	if proposal == nil {
		return "", fmt.Errorf("proposal with ID %s on chain %d not found", input.ProposalID, input.ChainID)
	}

	// Generate summary using AI
	summary, err := s.generateSummary(proposal.Description)
	if err != nil {
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}

	// Save to database
	now := time.Now()
	daoCode := input.DaoCode
	newSummary := dbmodels.ProposalSummary{
		ID:          utils.NextIDString(),
		DaoCode:     &daoCode,
		ChainId:     chainID,
		ProposalID:  input.ProposalID,
		Indexer:     &indexerEndpoint,
		Description: proposal.Description,
		Summary:     summary,
		CTime:       now,
		UTime:       &now,
	}

	slog.Info("[proposal-summary] Saving summary to database", "proposal_id", input.ProposalID, "dao_code", input.DaoCode)
	if err := s.db.Create(&newSummary).Error; err != nil {
		slog.Error("[proposal-summary] Failed to save summary to database", "error", err, "proposal_id", input.ProposalID)
		// Still return the generated summary even if saving fails
	} else {
		slog.Info("[proposal-summary] Summary saved successfully", "id", newSummary.ID, "proposal_id", input.ProposalID)
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
