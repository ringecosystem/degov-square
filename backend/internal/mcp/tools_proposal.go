package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ringecosystem/degov-square/services"
	"github.com/ringecosystem/degov-square/types"
	"gorm.io/gorm"
)

func addProposalTools(server *sdkmcp.Server) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "list_proposals",
		Title:       "List Proposals",
		Description: "Return bounded governance proposal rows for a DAO.",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, listProposalsTool)

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "get_proposal",
		Title:       "Get Proposal",
		Description: "Return a governance proposal detail payload for a DAO.",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, getProposalTool)
}

func listProposalsTool(ctx context.Context, req *sdkmcp.CallToolRequest, input listProposalsInput) (*sdkmcp.CallToolResult, listProposalsOutput, error) {
	daoCode, err := normalizeDaoCode(input.DaoCode)
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

func getProposalTool(ctx context.Context, req *sdkmcp.CallToolRequest, input getProposalInput) (*sdkmcp.CallToolResult, getProposalOutput, error) {
	daoCode, err := normalizeDaoCode(input.DaoCode)
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

	return nil, getProposalOutput{
		Proposal: proposalToolDTO(proposal),
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

func normalizeDaoCode(raw string) (string, error) {
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
