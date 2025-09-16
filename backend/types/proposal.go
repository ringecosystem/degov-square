package types

import (
	"time"

	dbmodels "github.com/ringecosystem/degov-square/database/models"
)

type ProposalTrackingInput struct {
	DaoCode           string
	ChainId           int
	Title             string
	ProposalLink      string
	ProposalID        string
	ProposalCreatedAt *time.Time
	ProposalAtBlock   int
}

type TrackingStateProposalsInput struct {
	DaoCode    string
	TimesTrack *int
	States     []dbmodels.ProposalState
}

// ProposalStateCountResult represents the result of proposal state count query
type ProposalStateCountResult struct {
	DaoCode string                 `json:"dao_code"`
	State   dbmodels.ProposalState `json:"state"`
	Total   int64                  `json:"total"`
}

type InspectProposalInput struct {
	DaoCode    string
	ProposalID string
}
