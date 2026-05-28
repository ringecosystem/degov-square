package mcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ringecosystem/degov-square/services"
	"github.com/ringecosystem/degov-square/types"
	"gorm.io/gorm"
)

func addProposalSummaryTool(server *sdkmcp.Server, cfg Config) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "summarize_proposal",
		Title:       "Summarize Proposal",
		Description: "Return a cached proposal summary, or generate one only when MCP summary generation is enabled.",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input summarizeProposalInput) (*sdkmcp.CallToolResult, summarizeProposalOutput, error) {
		return summarizeProposalTool(ctx, cfg, input)
	})
}

func summarizeProposalTool(ctx context.Context, cfg Config, input summarizeProposalInput) (*sdkmcp.CallToolResult, summarizeProposalOutput, error) {
	start := time.Now()
	cacheHit := false
	failureReason := ""
	defer func() {
		attrs := []any{
			"tool", "summarize_proposal",
			"duration_ms", time.Since(start).Milliseconds(),
			"cache_hit", cacheHit,
		}
		if failureReason != "" {
			attrs = append(attrs, "failure_reason", failureReason)
		}
		slog.Info("[mcp] tool completed", attrs...)
	}()

	daoCode, err := normalizeProposalDaoCode(input.DaoCode)
	if err != nil {
		failureReason = "invalid_dao_code"
		return nil, summarizeProposalOutput{}, err
	}
	proposalID := strings.TrimSpace(input.ProposalID)
	if proposalID == "" {
		failureReason = "invalid_proposal_id"
		return nil, summarizeProposalOutput{}, errors.New("invalid_proposal_id: proposalId is required")
	}

	serviceInput := services.ProposalSummaryInput{
		DaoCode:    daoCode,
		ProposalID: proposalID,
	}

	if !input.ForceRefresh {
		summary, err := cfg.ProposalSummaryService.GetCachedSummary(serviceInput)
		if err == nil && summary != nil {
			cacheHit = true
			generatedAt := summary.CTime
			if summary.UTime != nil {
				generatedAt = *summary.UTime
			}
			return nil, summarizeProposalOutput{
				DaoCode:     daoCode,
				ProposalID:  proposalID,
				Summary:     summary.Summary,
				Source:      "cache",
				CacheHit:    true,
				GeneratedAt: &generatedAt,
				Proposal:    lookupProposalMetadata(daoCode, proposalID),
			}, nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) && !isNotFoundError(err) {
			failureReason = "summary_cache_lookup_failed"
			return nil, summarizeProposalOutput{}, fmt.Errorf("summary_cache_lookup_failed: %w", err)
		}
	}

	if !cfg.ProposalSummaryGenerateEnabled {
		failureReason = "summary_unavailable"
		return nil, summarizeProposalOutput{}, errors.New("summary_unavailable: cached summary was not found and MCP summary generation is disabled")
	}

	summary, err := generateSummaryWithTimeout(ctx, cfg, serviceInput)
	if err != nil {
		failureReason = summaryFailureReason(err)
		return nil, summarizeProposalOutput{}, err
	}
	generatedAt := time.Now()
	return nil, summarizeProposalOutput{
		DaoCode:     daoCode,
		ProposalID:  proposalID,
		Summary:     summary,
		Source:      "generated",
		CacheHit:    false,
		GeneratedAt: &generatedAt,
		Proposal:    lookupProposalMetadata(daoCode, proposalID),
	}, nil
}

func generateSummaryWithTimeout(ctx context.Context, cfg Config, input services.ProposalSummaryInput) (string, error) {
	timeout := cfg.ProposalSummaryGenerationTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	type result struct {
		summary string
		err     error
	}
	resultCh := make(chan result, 1)
	go func() {
		summary, err := cfg.ProposalSummaryService.GetOrGenerateSummary(input)
		resultCh <- result{summary: summary, err: err}
	}()

	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("summary_generation_timeout: exceeded %s", timeout)
		}
		return "", fmt.Errorf("summary_generation_cancelled: %w", ctx.Err())
	case result := <-resultCh:
		if result.err != nil {
			return "", fmt.Errorf("summary_generation_failed: %w", result.err)
		}
		return result.summary, nil
	}
}

func summaryFailureReason(err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	switch {
	case strings.Contains(message, "summary_generation_timeout"):
		return "summary_generation_timeout"
	case strings.Contains(message, "summary_generation_cancelled"):
		return "summary_generation_cancelled"
	default:
		return "summary_generation_failed"
	}
}

func lookupProposalMetadata(daoCode, proposalID string) *proposalToolOutput {
	proposal, err := services.NewProposalService().InspectProposal(types.InspectProposalInput{
		DaoCode:    daoCode,
		ProposalID: proposalID,
	})
	if err != nil || proposal == nil {
		return nil
	}
	output := proposalToolDTO(proposal)
	return &output
}
