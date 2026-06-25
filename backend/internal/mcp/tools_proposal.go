package mcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	dbmodels "github.com/ringecosystem/degov-square/database/models"
	gqlmodels "github.com/ringecosystem/degov-square/graph/models"
	degovinternal "github.com/ringecosystem/degov-square/internal"
	"github.com/ringecosystem/degov-square/services"
	"github.com/ringecosystem/degov-square/types"
	"gorm.io/gorm"
)

func addProposalTools(server *sdkmcp.Server, cfg Config) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "list_proposals",
		Title:       "List Proposals",
		Description: "Return bounded governance proposal rows for a DAO.",
		Annotations: readOnlyToolAnnotations(),
		InputSchema: listProposalsInputSchema(),
	}, listProposalsTool)

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "get_proposal",
		Title:       "Get Proposal",
		Description: "Return a governance proposal detail payload for a DAO.",
		Annotations: readOnlyToolAnnotations(),
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input getProposalInput) (*sdkmcp.CallToolResult, getProposalOutput, error) {
		return getProposalTool(ctx, cfg, input)
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "get_proposal_state",
		Title:       "Get Proposal State",
		Description: "Return the cached governance proposal state for a DAO.",
		Annotations: readOnlyToolAnnotations(),
	}, getProposalStateTool)
}

func listProposalsInputSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:     "object",
		Required: []string{"daoCode"},
		Properties: map[string]*jsonschema.Schema{
			"daoCode": {
				Type:        "string",
				Description: "DAO code, for example ring-dao.",
			},
			"state": {
				Type:        "string",
				Description: "Optional proposal state filter.",
				Enum: []any{
					string(dbmodels.ProposalStateUnknown),
					string(dbmodels.ProposalStatePending),
					string(dbmodels.ProposalStateActive),
					string(dbmodels.ProposalStateCanceled),
					string(dbmodels.ProposalStateDefeated),
					string(dbmodels.ProposalStateSucceeded),
					string(dbmodels.ProposalStateQueued),
					string(dbmodels.ProposalStateExecuted),
					string(dbmodels.ProposalStateExpired),
					strings.ToLower(string(dbmodels.ProposalStateUnknown)),
					strings.ToLower(string(dbmodels.ProposalStatePending)),
					strings.ToLower(string(dbmodels.ProposalStateActive)),
					strings.ToLower(string(dbmodels.ProposalStateCanceled)),
					strings.ToLower(string(dbmodels.ProposalStateDefeated)),
					strings.ToLower(string(dbmodels.ProposalStateSucceeded)),
					strings.ToLower(string(dbmodels.ProposalStateQueued)),
					strings.ToLower(string(dbmodels.ProposalStateExecuted)),
					strings.ToLower(string(dbmodels.ProposalStateExpired)),
				},
			},
			"limit": {
				Type:        "integer",
				Description: "Maximum number of proposals to return.",
			},
			"offset": {
				Type:        "integer",
				Description: "Number of proposals to skip.",
			},
		},
	}
}

func listProposalsTool(ctx context.Context, req *sdkmcp.CallToolRequest, input listProposalsInput) (*sdkmcp.CallToolResult, listProposalsOutput, error) {
	daoCode, err := normalizeProposalDaoCode(input.DaoCode)
	if err != nil {
		return nil, listProposalsOutput{}, err
	}
	if input.Offset < 0 {
		return nil, listProposalsOutput{}, errors.New("invalid_offset: offset must be greater than or equal to 0")
	}

	limit := normalizeProposalListLimit(input.Limit)
	var state string
	listInput := types.ListProposalsInput{
		DaoCode: daoCode,
		Limit:   limit,
		Offset:  input.Offset,
	}
	if strings.TrimSpace(input.State) != "" {
		normalizedState, err := normalizeProposalState(input.State)
		if err != nil {
			return nil, listProposalsOutput{}, err
		}
		listInput.State = normalizedState
		state = string(normalizedState)
	}

	if err := requireDAO(daoCode); err != nil {
		return nil, listProposalsOutput{}, err
	}

	proposalService := services.NewProposalService()
	proposals, err := proposalService.ListProposals(listInput)
	if err != nil {
		return nil, listProposalsOutput{}, fmt.Errorf("proposal_list_failed: %w", err)
	}

	output := listProposalsOutput{
		DaoCode:   daoCode,
		State:     state,
		Limit:     limit,
		Offset:    input.Offset,
		Proposals: make([]proposalToolOutput, 0, len(proposals)),
	}
	for _, proposal := range proposals {
		output.Proposals = append(output.Proposals, proposalToolDTO(proposal))
	}

	return nil, output, nil
}

func getProposalTool(ctx context.Context, cfg Config, input getProposalInput) (*sdkmcp.CallToolResult, getProposalOutput, error) {
	daoCode, err := normalizeProposalDaoCode(input.DaoCode)
	if err != nil {
		return nil, getProposalOutput{}, err
	}
	proposalID := strings.TrimSpace(input.ProposalID)
	if proposalID == "" {
		return nil, getProposalOutput{}, errors.New("invalid_proposal_id: proposalId is required")
	}
	if err := requireDAO(daoCode); err != nil {
		return nil, getProposalOutput{}, err
	}

	proposalService := services.NewProposalService()
	proposal, err := proposalService.InspectProposal(types.InspectProposalInput{
		DaoCode:    daoCode,
		ProposalID: proposalID,
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, getProposalOutput{}, fmt.Errorf("proposal_not_found: proposal %q was not found for DAO %q", proposalID, daoCode)
		}
		return nil, getProposalOutput{}, fmt.Errorf("proposal_lookup_failed: %w", err)
	}

	output := proposalToolDTO(proposal)
	if indexerProposal, err := inspectMCPIndexerProposal(ctx, cfg, daoCode, proposalID); err == nil && indexerProposal != nil {
		output.Proposer = proposerIdentityOutput(ctx, cfg, daoCode, indexerProposal.Proposer)
	} else if err != nil {
		slog.Debug("MCP proposal proposer enrichment failed", "daoCode", daoCode, "proposalId", proposalID, "err", err)
	}

	return nil, getProposalOutput{
		Proposal: output,
	}, nil
}

func inspectMCPIndexerProposal(ctx context.Context, cfg Config, daoCode string, proposalID string) (*degovinternal.Proposal, error) {
	indexer, scope, err := mcpIndexerForDAO(daoCode, cfg)
	if err != nil {
		return nil, err
	}
	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return indexer.InspectProposalWithContext(queryCtx, scope, proposalID)
}

func proposerIdentityOutput(ctx context.Context, cfg Config, daoCode string, address string) *addressIdentityOutput {
	if strings.TrimSpace(address) == "" {
		return nil
	}
	identities := hydrateAddressIdentities(ctx, cfg, daoCode, []string{address})
	identity := addressIdentityFromMap(identities, address)
	return &identity
}

func getProposalStateTool(ctx context.Context, req *sdkmcp.CallToolRequest, input getProposalStateInput) (*sdkmcp.CallToolResult, getProposalStateOutput, error) {
	daoCode, err := normalizeProposalDaoCode(input.DaoCode)
	if err != nil {
		return nil, getProposalStateOutput{}, err
	}
	proposalID := strings.TrimSpace(input.ProposalID)
	if proposalID == "" {
		return nil, getProposalStateOutput{}, errors.New("invalid_proposal_id: proposalId is required")
	}
	if err := requireDAO(daoCode); err != nil {
		return nil, getProposalStateOutput{}, err
	}

	proposalService := services.NewProposalService()
	state, err := proposalService.GetProposalState(gqlmodels.ProposalStateInput{
		DaoCode:    daoCode,
		ProposalID: proposalID,
	})
	if err != nil {
		return nil, getProposalStateOutput{}, fmt.Errorf("proposal_state_lookup_failed: %w", err)
	}
	if state == nil {
		return nil, getProposalStateOutput{}, fmt.Errorf("proposal_not_found: proposal %q was not found for DAO %q", proposalID, daoCode)
	}
	if strings.TrimSpace(string(*state)) == "" {
		return nil, getProposalStateOutput{}, fmt.Errorf("proposal_state_unavailable: proposal %q has no cached state for DAO %q", proposalID, daoCode)
	}

	proposal, err := proposalService.InspectProposal(types.InspectProposalInput{
		DaoCode:    daoCode,
		ProposalID: proposalID,
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, getProposalStateOutput{}, fmt.Errorf("proposal_not_found: proposal %q was not found for DAO %q", proposalID, daoCode)
		}
		return nil, getProposalStateOutput{}, fmt.Errorf("proposal_lookup_failed: %w", err)
	}

	updatedAt := proposal.CTime
	if proposal.UTime != nil {
		updatedAt = *proposal.UTime
	}
	return nil, getProposalStateOutput{
		DaoCode:           daoCode,
		ProposalID:        proposalID,
		State:             string(*state),
		Source:            "tracked",
		ProposalCreatedAt: proposal.ProposalCreatedAt,
		UpdatedAt:         &updatedAt,
	}, nil
}

func normalizeProposalListLimit(limit int) int {
	if limit <= 0 {
		return defaultProposalListLimit
	}
	if limit > maxProposalListLimit {
		return maxProposalListLimit
	}
	return limit
}

func normalizeProposalDaoCode(raw string) (string, error) {
	daoCode := strings.TrimSpace(raw)
	if daoCode == "" {
		return "", errors.New("invalid_dao_code: daoCode is required")
	}
	return daoCode, nil
}

func requireDAO(daoCode string) error {
	daoService := services.NewDaoService()
	dao, err := daoService.GetByCode(daoCode)
	if err != nil {
		return fmt.Errorf("dao_lookup_failed: %w", err)
	}
	if dao == nil {
		return fmt.Errorf("dao_not_found: DAO %q was not found", daoCode)
	}
	return nil
}
