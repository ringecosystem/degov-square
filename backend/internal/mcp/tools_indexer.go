package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	gqlmodels "github.com/ringecosystem/degov-square/graph/models"
	degovinternal "github.com/ringecosystem/degov-square/internal"
	"github.com/ringecosystem/degov-square/types"
	"gopkg.in/yaml.v3"
)

func addIndexerTools(server *sdkmcp.Server, cfg Config) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "get_contributor",
		Title:       "Get Contributor",
		Description: "Return an indexer-backed governance contributor for a DAO.",
		Annotations: readOnlyToolAnnotations(),
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input getContributorInput) (*sdkmcp.CallToolResult, getContributorOutput, error) {
		return getContributorTool(ctx, cfg, input)
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "list_contributors",
		Title:       "List Contributors",
		Description: "Return bounded indexer-backed governance contributors for a DAO.",
		Annotations: readOnlyToolAnnotations(),
		InputSchema: listContributorsInputSchema(),
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input listContributorsInput) (*sdkmcp.CallToolResult, listContributorsOutput, error) {
		return listContributorsTool(ctx, cfg, input)
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "list_proposal_votes",
		Title:       "List Proposal Votes",
		Description: "Return bounded indexer-backed vote rows for a DAO proposal.",
		Annotations: readOnlyToolAnnotations(),
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input listProposalVotesInput) (*sdkmcp.CallToolResult, listProposalVotesOutput, error) {
		return listProposalVotesTool(ctx, cfg, input)
	})
}

func listContributorsInputSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:     "object",
		Required: []string{"daoCode"},
		Properties: map[string]*jsonschema.Schema{
			"daoCode": {
				Type:        "string",
				Description: "DAO code, for example ring-dao.",
				Pattern:     daoCodePattern.String(),
			},
			"limit": {
				Type:        "integer",
				Description: "Maximum number of contributors to return.",
			},
			"offset": {
				Type:        "integer",
				Description: "Number of contributors to skip.",
			},
			"orderBy": {
				Type:        "string",
				Description: "Contributor sort order. Defaults to power_desc.",
				Enum:        []any{"power_desc", "power_asc", "id_asc"},
			},
		},
	}
}

func getContributorTool(ctx context.Context, cfg Config, input getContributorInput) (*sdkmcp.CallToolResult, getContributorOutput, error) {
	daoCode, err := normalizeProposalDaoCode(input.DaoCode)
	if err != nil {
		return nil, getContributorOutput{}, err
	}
	address, err := normalizeENSAddress(input.Address)
	if err != nil {
		return nil, getContributorOutput{}, fmt.Errorf("invalid_address: %w", err)
	}
	indexer, scope, err := mcpIndexerForDAO(daoCode, cfg)
	if err != nil {
		return nil, getContributorOutput{}, err
	}

	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	contributor, err := indexer.QueryContributor(queryCtx, scope, address)
	if err != nil {
		return nil, getContributorOutput{}, fmt.Errorf("contributor_lookup_failed: %w", err)
	}

	identities := hydrateAddressIdentities(ctx, cfg, daoCode, []string{contributor.ID})
	return nil, getContributorOutput{
		Contributor: contributorToolDTO(*contributor, identities),
	}, nil
}

func listContributorsTool(ctx context.Context, cfg Config, input listContributorsInput) (*sdkmcp.CallToolResult, listContributorsOutput, error) {
	daoCode, err := normalizeProposalDaoCode(input.DaoCode)
	if err != nil {
		return nil, listContributorsOutput{}, err
	}
	if input.Offset < 0 {
		return nil, listContributorsOutput{}, errors.New("invalid_offset: offset must be greater than or equal to 0")
	}
	limit := normalizeContributorListLimit(input.Limit)
	orderBy, err := normalizeContributorOrder(input.OrderBy)
	if err != nil {
		return nil, listContributorsOutput{}, err
	}
	indexer, scope, err := mcpIndexerForDAO(daoCode, cfg)
	if err != nil {
		return nil, listContributorsOutput{}, err
	}

	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	contributors, err := indexer.QueryContributors(queryCtx, scope, input.Offset, limit, orderBy)
	if err != nil {
		return nil, listContributorsOutput{}, fmt.Errorf("contributor_list_failed: %w", err)
	}

	addresses := make([]string, 0, len(contributors))
	for _, contributor := range contributors {
		addresses = append(addresses, contributor.ID)
	}
	identities := hydrateAddressIdentities(ctx, cfg, daoCode, addresses)

	output := listContributorsOutput{
		DaoCode:      daoCode,
		Limit:        limit,
		Offset:       input.Offset,
		Contributors: make([]contributorOutput, 0, len(contributors)),
	}
	for _, contributor := range contributors {
		output.Contributors = append(output.Contributors, contributorToolDTO(contributor, identities))
	}
	return nil, output, nil
}

func listProposalVotesTool(ctx context.Context, cfg Config, input listProposalVotesInput) (*sdkmcp.CallToolResult, listProposalVotesOutput, error) {
	daoCode, err := normalizeProposalDaoCode(input.DaoCode)
	if err != nil {
		return nil, listProposalVotesOutput{}, err
	}
	proposalID := strings.TrimSpace(input.ProposalID)
	if proposalID == "" {
		return nil, listProposalVotesOutput{}, errors.New("invalid_proposal_id: proposalId is required")
	}
	if input.Offset < 0 {
		return nil, listProposalVotesOutput{}, errors.New("invalid_offset: offset must be greater than or equal to 0")
	}
	limit := normalizeProposalVotesListLimit(input.Limit)
	indexer, scope, err := mcpIndexerForDAO(daoCode, cfg)
	if err != nil {
		return nil, listProposalVotesOutput{}, err
	}

	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	votes, err := indexer.QueryVotes(queryCtx, scope, input.Offset, limit, proposalID)
	if err != nil {
		return nil, listProposalVotesOutput{}, fmt.Errorf("proposal_votes_list_failed: %w", err)
	}

	addresses := make([]string, 0, len(votes))
	for _, vote := range votes {
		addresses = append(addresses, vote.Voter)
	}
	identities := hydrateAddressIdentities(ctx, cfg, daoCode, addresses)

	output := listProposalVotesOutput{
		DaoCode:    daoCode,
		ProposalID: proposalID,
		Limit:      limit,
		Offset:     input.Offset,
		Votes:      make([]voteOutput, 0, len(votes)),
	}
	for _, vote := range votes {
		output.Votes = append(output.Votes, voteToolDTO(vote, identities))
	}
	return nil, output, nil
}

func mcpIndexerForDAO(daoCode string, cfg Config) (*degovinternal.DegovIndexer, degovinternal.ProposalScope, error) {
	if err := requireDAO(daoCode); err != nil {
		return nil, degovinternal.ProposalScope{}, err
	}
	cfg = withDefaultDaoServices(cfg)
	rawConfig, err := cfg.DaoConfigService.RawConfig(gqlmodels.GetDaoConfigInput{
		DaoCode: daoCode,
	})
	if err != nil {
		return nil, degovinternal.ProposalScope{}, fmt.Errorf("dao_config_lookup_failed: %w", err)
	}

	var daoConfig types.DaoConfig
	if err := yaml.Unmarshal([]byte(rawConfig), &daoConfig); err != nil {
		return nil, degovinternal.ProposalScope{}, fmt.Errorf("dao_config_parse_failed: %w", err)
	}
	if strings.TrimSpace(daoConfig.Indexer.Endpoint) == "" {
		return nil, degovinternal.ProposalScope{}, fmt.Errorf("dao_indexer_unavailable: DAO %q has no indexer endpoint", daoCode)
	}

	return degovinternal.NewDegovIndexer(daoConfig.Indexer.Endpoint), degovinternal.ProposalScope{
		ChainID:         daoConfig.Chain.ID,
		DaoCode:         daoCode,
		GovernorAddress: daoConfig.Contracts.Governor,
	}, nil
}

func normalizeContributorListLimit(limit int) int {
	if limit <= 0 {
		return defaultContributorListLimit
	}
	if limit > maxContributorListLimit {
		return maxContributorListLimit
	}
	return limit
}

func normalizeProposalVotesListLimit(limit int) int {
	if limit <= 0 {
		return defaultProposalVotesListLimit
	}
	if limit > maxProposalVotesListLimit {
		return maxProposalVotesListLimit
	}
	return limit
}

func normalizeContributorOrder(orderBy string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(orderBy)) {
	case "", "power_desc":
		return "power_DESC", nil
	case "power_asc":
		return "power_ASC", nil
	case "id_asc":
		return "id_ASC", nil
	default:
		return "", errors.New("invalid_order_by: orderBy must be power_desc, power_asc, or id_asc")
	}
}

func contributorToolDTO(contributor degovinternal.Contributor, identities map[string]addressIdentityOutput) contributorOutput {
	balance := ""
	if contributor.Balance != nil {
		balance = *contributor.Balance
	}
	return contributorOutput{
		Account:                 addressIdentityFromMap(identities, contributor.ID),
		Power:                   contributor.Power,
		Balance:                 balance,
		DelegatesCountAll:       contributor.DelegatesCountAll,
		DelegatesCountEffective: contributor.DelegatesCountEffective,
	}
}

func voteToolDTO(vote degovinternal.VoteCast, identities map[string]addressIdentityOutput) voteOutput {
	return voteOutput{
		ID:              vote.ID,
		ProposalID:      vote.ProposalID,
		Voter:           addressIdentityFromMap(identities, vote.Voter),
		Support:         vote.Support,
		Weight:          vote.Weight,
		Reason:          vote.Reason,
		TransactionHash: vote.TransactionHash,
		BlockNumber:     vote.BlockNumber,
		BlockTimestamp:  vote.BlockTimestamp,
	}
}
