package mcp

import (
	"errors"
	"testing"
	"time"

	"github.com/ringecosystem/degov-square/database"
	dbmodels "github.com/ringecosystem/degov-square/database/models"
	"github.com/ringecosystem/degov-square/services"
)

type fakeProposalSummaryService struct {
	cachedSummary    *dbmodels.ProposalSummary
	cachedErr        error
	generatedSummary string
	generateErr      error
	generateDelay    time.Duration
	generateInput    services.ProposalSummaryInput
	generateCalls    int
}

func (s *fakeProposalSummaryService) GetCachedSummary(input services.ProposalSummaryInput) (*dbmodels.ProposalSummary, error) {
	return s.cachedSummary, s.cachedErr
}

func (s *fakeProposalSummaryService) GetOrGenerateSummary(input services.ProposalSummaryInput) (string, error) {
	s.generateInput = input
	s.generateCalls++
	if s.generateDelay > 0 {
		time.Sleep(s.generateDelay)
	}
	return s.generatedSummary, s.generateErr
}

func TestSummarizeProposalToolReturnsCachedSummary(t *testing.T) {
	server := newTestProposalServer(t)
	generatedAt := time.Now().Add(-time.Hour).UTC()
	seedMCPProposal(t, dbmodels.ProposalTracking{
		ID:           "proposal-cached",
		DaoCode:      "ring-dao",
		ChainId:      46,
		Title:        "Cached proposal",
		ProposalLink: "https://gov.ringdao.com/proposal/cached",
		ProposalID:   "0xcached",
		State:        dbmodels.ProposalStateExecuted,
		CTime:        time.Now(),
	})
	seedMCPProposalSummary(t, dbmodels.ProposalSummary{
		ID:          "summary-cached",
		DaoCode:     stringPtr("ring-dao"),
		ChainId:     46,
		ProposalID:  "0xcached",
		Description: "Proposal description",
		Summary:     "Cached proposal summary",
		CTime:       generatedAt,
		UTime:       &generatedAt,
	})

	result := callProposalTool(t, server, "summarize_proposal", map[string]any{
		"daoCode":    "ring-dao",
		"proposalId": "0xcached",
	})
	content := requireStructuredContent(t, result)

	if got, want := content["summary"], "Cached proposal summary"; got != want {
		t.Fatalf("summary = %v, want %v", got, want)
	}
	if got, want := content["source"], "cache"; got != want {
		t.Fatalf("source = %v, want %v", got, want)
	}
	if got, want := content["cacheHit"], true; got != want {
		t.Fatalf("cacheHit = %v, want %v", got, want)
	}
	proposal := content["proposal"].(map[string]any)
	if got, want := proposal["proposalId"], "0xcached"; got != want {
		t.Fatalf("proposalId = %v, want %v", got, want)
	}
}

func TestSummarizeProposalToolRejectsGenerationByDefault(t *testing.T) {
	server := NewServer(Config{
		Name:                   "degov-square",
		Version:                "test-version",
		ProposalSummaryService: &fakeProposalSummaryService{cachedErr: errors.New("record not found")},
	})

	result := callProposalTool(t, server, "summarize_proposal", map[string]any{
		"daoCode":    "ring-dao",
		"proposalId": "0xmissing",
	})

	requireToolErrorContains(t, result, "summary_unavailable")
}

func TestSummarizeProposalToolGeneratesWhenEnabled(t *testing.T) {
	server := newTestProposalServer(t)
	summaryService := &fakeProposalSummaryService{
		cachedErr:        errors.New("record not found"),
		generatedSummary: "Generated proposal summary",
	}
	server = NewServer(Config{
		Name:                             "degov-square",
		Version:                          "test-version",
		ProposalSummaryGenerateEnabled:   true,
		ProposalSummaryGenerationTimeout: time.Second,
		ProposalSummaryService:           summaryService,
	})
	seedMCPProposal(t, dbmodels.ProposalTracking{
		ID:           "proposal-generated",
		DaoCode:      "ring-dao",
		ChainId:      46,
		Title:        "Generated proposal",
		ProposalLink: "https://gov.ringdao.com/proposal/generated",
		ProposalID:   "0xgenerated",
		State:        dbmodels.ProposalStateActive,
		CTime:        time.Now(),
	})

	result := callProposalTool(t, server, "summarize_proposal", map[string]any{
		"daoCode":      "ring-dao",
		"proposalId":   "0xgenerated",
		"forceRefresh": true,
	})
	content := requireStructuredContent(t, result)

	if got, want := content["summary"], "Generated proposal summary"; got != want {
		t.Fatalf("summary = %v, want %v", got, want)
	}
	if got, want := content["source"], "generated"; got != want {
		t.Fatalf("source = %v, want %v", got, want)
	}
	if got, want := content["cacheHit"], false; got != want {
		t.Fatalf("cacheHit = %v, want %v", got, want)
	}
	if got, want := summaryService.generateInput.ProposalID, "0xgenerated"; got != want {
		t.Fatalf("generated proposal id = %q, want %q", got, want)
	}
}

func TestSummarizeProposalToolBoundsGenerationByTimeout(t *testing.T) {
	server := NewServer(Config{
		Name:                             "degov-square",
		Version:                          "test-version",
		ProposalSummaryGenerateEnabled:   true,
		ProposalSummaryGenerationTimeout: 10 * time.Millisecond,
		ProposalSummaryService: &fakeProposalSummaryService{
			cachedErr:     errors.New("record not found"),
			generateDelay: 100 * time.Millisecond,
		},
	})

	result := callProposalTool(t, server, "summarize_proposal", map[string]any{
		"daoCode":    "ring-dao",
		"proposalId": "0xslow",
	})

	requireToolErrorContains(t, result, "summary_generation_timeout")
}

func TestSummarizeProposalToolReturnsStructuredFailure(t *testing.T) {
	server := NewServer(Config{
		Name:                             "degov-square",
		Version:                          "test-version",
		ProposalSummaryGenerateEnabled:   true,
		ProposalSummaryGenerationTimeout: time.Second,
		ProposalSummaryService: &fakeProposalSummaryService{
			cachedErr:   errors.New("record not found"),
			generateErr: errors.New("openrouter failed"),
		},
	})

	result := callProposalTool(t, server, "summarize_proposal", map[string]any{
		"daoCode":    "ring-dao",
		"proposalId": "0xfail",
	})

	requireToolErrorContains(t, result, "summary_generation_failed")
}

func TestSummarizeProposalToolRejectsInvalidInput(t *testing.T) {
	server := NewServer(Config{Name: "degov-square", Version: "test-version"})

	result := callProposalTool(t, server, "summarize_proposal", map[string]any{
		"daoCode":    "",
		"proposalId": "0xproposal",
	})

	requireToolErrorContains(t, result, "invalid_dao_code")
}

func stringPtr(value string) *string {
	return &value
}

func seedMCPProposalSummary(t *testing.T, summary dbmodels.ProposalSummary) {
	t.Helper()

	if err := database.DB.Create(&summary).Error; err != nil {
		t.Fatalf("seed proposal summary: %v", err)
	}
}
