package types

import (
	"time"

	dbmodels "github.com/ringecosystem/degov-apps/database/models"
)

type ProposalTrackingInput struct {
	DaoCode           string
	ChainId           int
	ProposalLink      string
	ProposalID        string
	ProposalCreatedAt *time.Time
	ProposalAtBlock   int
}

type TrackingStateProposalsInput struct {
	DaoCode string
	States  []dbmodels.ProposalState
}
