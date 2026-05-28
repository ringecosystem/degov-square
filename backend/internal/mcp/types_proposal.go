package mcp

import (
	"fmt"
	"strings"
	"time"

	dbmodels "github.com/ringecosystem/degov-square/database/models"
)

const (
	defaultProposalListLimit = 20
	maxProposalListLimit     = 50
)

type listProposalsInput struct {
	DaoCode string `json:"daoCode" jsonschema:"DAO code"`
	State   string `json:"state,omitempty" jsonschema:"Optional proposal state filter"`
	Limit   int    `json:"limit,omitempty" jsonschema:"Maximum rows to return"`
	Offset  int    `json:"offset,omitempty" jsonschema:"Rows to skip"`
}

type getProposalInput struct {
	DaoCode    string `json:"daoCode" jsonschema:"DAO code"`
	ProposalID string `json:"proposalId" jsonschema:"Proposal id"`
}

type listProposalsOutput struct {
	DaoCode   string               `json:"daoCode"`
	State     string               `json:"state,omitempty"`
	Limit     int                  `json:"limit"`
	Offset    int                  `json:"offset"`
	Proposals []proposalToolOutput `json:"proposals"`
}

type getProposalOutput struct {
	Proposal proposalToolOutput `json:"proposal"`
}

type proposalToolOutput struct {
	ID                 string     `json:"id"`
	DaoCode            string     `json:"daoCode"`
	ChainID            int        `json:"chainId"`
	Title              string     `json:"title"`
	ProposalLink       string     `json:"proposalLink"`
	ProposalID         string     `json:"proposalId"`
	State              string     `json:"state"`
	ProposalCreatedAt  *time.Time `json:"proposalCreatedAt,omitempty"`
	ProposalAtBlock    int        `json:"proposalAtBlock"`
	OffsetTrackingVote int        `json:"offsetTrackingVote"`
	Fulfilled          int        `json:"fulfilled"`
	FulfilledExplain   *string    `json:"fulfilledExplain,omitempty"`
	FulfilledAt        *time.Time `json:"fulfilledAt,omitempty"`
	CTime              time.Time  `json:"ctime"`
	UTime              *time.Time `json:"utime,omitempty"`
}

func normalizeProposalState(raw string) (dbmodels.ProposalState, error) {
	state := dbmodels.ProposalState(strings.ToUpper(strings.TrimSpace(raw)))
	switch state {
	case dbmodels.ProposalStateUnknown,
		dbmodels.ProposalStatePending,
		dbmodels.ProposalStateActive,
		dbmodels.ProposalStateCanceled,
		dbmodels.ProposalStateDefeated,
		dbmodels.ProposalStateSucceeded,
		dbmodels.ProposalStateQueued,
		dbmodels.ProposalStateExecuted,
		dbmodels.ProposalStateExpired:
		return state, nil
	default:
		return "", fmt.Errorf("invalid_state: %q is not a valid proposal state", raw)
	}
}

func proposalToolDTO(proposal *dbmodels.ProposalTracking) proposalToolOutput {
	return proposalToolOutput{
		ID:                 proposal.ID,
		DaoCode:            proposal.DaoCode,
		ChainID:            proposal.ChainId,
		Title:              proposal.Title,
		ProposalLink:       proposal.ProposalLink,
		ProposalID:         proposal.ProposalID,
		State:              string(proposal.State),
		ProposalCreatedAt:  proposal.ProposalCreatedAt,
		ProposalAtBlock:    proposal.ProposalAtBlock,
		OffsetTrackingVote: proposal.OffsetTrackingVote,
		Fulfilled:          proposal.Fulfilled,
		FulfilledExplain:   proposal.FulfilledExplain,
		FulfilledAt:        proposal.FulfilledAt,
		CTime:              proposal.CTime,
		UTime:              proposal.UTime,
	}
}
