package types

import "time"


type ProposalTrackingInput struct {
	DaoCode           string
	ProposalLink      string
	ProposalID        string
	ProposalCreatedAt *time.Time
	ProposalAtBlock   int
}
