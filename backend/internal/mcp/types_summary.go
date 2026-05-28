package mcp

import "time"

type summarizeProposalInput struct {
	DaoCode      string `json:"daoCode" jsonschema:"DAO code"`
	ProposalID   string `json:"proposalId" jsonschema:"Proposal id"`
	ForceRefresh bool   `json:"forceRefresh,omitempty" jsonschema:"Generate a fresh summary when generation is enabled"`
}

type summarizeProposalOutput struct {
	DaoCode     string              `json:"daoCode"`
	ProposalID  string              `json:"proposalId"`
	Summary     string              `json:"summary"`
	Source      string              `json:"source"`
	CacheHit    bool                `json:"cacheHit"`
	GeneratedAt *time.Time          `json:"generatedAt,omitempty"`
	Proposal    *proposalToolOutput `json:"proposal,omitempty"`
}
