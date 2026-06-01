package mcp

const (
	defaultContributorListLimit   = 20
	maxContributorListLimit       = 50
	defaultProposalVotesListLimit = 20
	maxProposalVotesListLimit     = 50
)

type getContributorInput struct {
	DaoCode string `json:"daoCode" jsonschema:"DAO code"`
	Address string `json:"address" jsonschema:"Contributor address"`
}

type listContributorsInput struct {
	DaoCode string `json:"daoCode" jsonschema:"DAO code"`
	Limit   int    `json:"limit,omitempty" jsonschema:"Maximum rows to return"`
	Offset  int    `json:"offset,omitempty" jsonschema:"Rows to skip"`
	OrderBy string `json:"orderBy,omitempty" jsonschema:"Optional order: power_desc, power_asc, or id_asc"`
}

type listProposalVotesInput struct {
	DaoCode    string `json:"daoCode" jsonschema:"DAO code"`
	ProposalID string `json:"proposalId" jsonschema:"Proposal id"`
	Limit      int    `json:"limit,omitempty" jsonschema:"Maximum rows to return"`
	Offset     int    `json:"offset,omitempty" jsonschema:"Rows to skip"`
}

type getContributorOutput struct {
	Contributor contributorOutput `json:"contributor"`
}

type listContributorsOutput struct {
	DaoCode      string              `json:"daoCode"`
	Limit        int                 `json:"limit"`
	Offset       int                 `json:"offset"`
	Contributors []contributorOutput `json:"contributors"`
}

type listProposalVotesOutput struct {
	DaoCode    string       `json:"daoCode"`
	ProposalID string       `json:"proposalId"`
	Limit      int          `json:"limit"`
	Offset     int          `json:"offset"`
	Votes      []voteOutput `json:"votes"`
}

type contributorOutput struct {
	Account                 addressIdentityOutput `json:"account"`
	Power                   string                `json:"power"`
	Balance                 string                `json:"balance"`
	DelegatesCountAll       int                   `json:"delegatesCountAll"`
	DelegatesCountEffective int                   `json:"delegatesCountEffective"`
}

type voteOutput struct {
	ID              string                `json:"id"`
	ProposalID      string                `json:"proposalId"`
	Voter           addressIdentityOutput `json:"voter"`
	Support         int                   `json:"support"`
	Weight          string                `json:"weight"`
	Reason          string                `json:"reason"`
	TransactionHash string                `json:"transactionHash"`
	BlockNumber     string                `json:"blockNumber"`
	BlockTimestamp  string                `json:"blockTimestamp"`
}
