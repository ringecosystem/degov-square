package internal

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/machinebox/graphql"
)

// DataMetrics represents the data metrics structure from GraphQL response
type DataMetrics struct {
	ProposalsCount          *int   `json:"proposalsCount"`
	MemberCount             *int   `json:"memberCount"`
	HoldersCount            *int   `json:"holdersCount"`
	ContributorCount        *int   `json:"contributorCount"`
	PowerSum                string `json:"powerSum"`
	VotesCount              int    `json:"votesCount"`
	VotesWeightAbstainSum   string `json:"votesWeightAbstainSum"`
	VotesWeightAgainstSum   string `json:"votesWeightAgainstSum"`
	VotesWeightForSum       string `json:"votesWeightForSum"`
	VotesWithParamsCount    int    `json:"votesWithParamsCount"`
	VotesWithoutParamsCount int    `json:"votesWithoutParamsCount"`
	ID                      string `json:"id"`
}

func (m DataMetrics) MemberCountValue() *int {
	if m.HoldersCount != nil {
		return m.HoldersCount
	}
	return m.MemberCount
}

// DataMetricsResponse represents the GraphQL response structure
type DataMetricsResponse struct {
	DataMetrics []DataMetrics `json:"dataMetrics"`
}

type ProposalPage struct {
	TotalCount int `json:"totalCount"`
}

type ProposalPageResponse struct {
	ProposalsPage ProposalPage `json:"proposalsPage"`
}

// Proposal represents the structure of a governance proposal.
type Proposal struct {
	ID                             string  `json:"id"`
	ChainID                        *int    `json:"chainId"`
	DaoCode                        string  `json:"daoCode"`
	GovernorAddress                string  `json:"governorAddress"`
	ProposalID                     string  `json:"proposalId"`
	Title                          string  `json:"title"`
	Quorum                         string  `json:"quorum"`
	VoteStartTimestamp             string  `json:"voteStartTimestamp"`
	VoteEndTimestamp               string  `json:"voteEndTimestamp"`
	VoteStart                      string  `json:"voteStart"`
	VoteEnd                        string  `json:"voteEnd"`
	Decimals                       string  `json:"decimals"`
	BlockInterval                  string  `json:"blockInterval"`
	ClockMode                      string  `json:"clockMode"`
	Proposer                       string  `json:"proposer"`
	BlockNumber                    string  `json:"blockNumber"`
	BlockTimestamp                 string  `json:"blockTimestamp"`
	TransactionHash                string  `json:"transactionHash"`
	ProposalDeadline               string  `json:"proposalDeadline"`
	ProposalEta                    string  `json:"proposalEta"`
	QueueReadyAt                   string  `json:"queueReadyAt"`
	QueueExpiresAt                 string  `json:"queueExpiresAt"`
	TimelockAddress                string  `json:"timelockAddress"`
	TimelockGracePeriod            string  `json:"timelockGracePeriod"`
	MetricsVotesCount              *int    `json:"metricsVotesCount"`
	MetricsVotesWithParamsCount    *int    `json:"metricsVotesWithParamsCount"`
	MetricsVotesWithoutParamsCount *int    `json:"metricsVotesWithoutParamsCount"`
	MetricsVotesWeightAbstainSum   *string `json:"metricsVotesWeightAbstainSum"`
	MetricsVotesWeightAgainstSum   *string `json:"metricsVotesWeightAgainstSum"`
	MetricsVotesWeightForSum       *string `json:"metricsVotesWeightForSum"`
	Description                    string  `json:"description"`
}

// ProposalsResponse represents the GraphQL response structure for proposals
type ProposalsResponse struct {
	Proposals []Proposal `json:"proposals"`
}

type VoteCast struct {
	ProposalID      string `json:"proposalId"`
	Reason          string `json:"reason"`
	Support         int    `json:"support"`
	Voter           string `json:"voter"`
	Weight          string `json:"weight"`
	TransactionHash string `json:"transactionHash"`
	ID              string `json:"id"`
	BlockNumber     string `json:"blockNumber"`
	BlockTimestamp  string `json:"blockTimestamp"`
}

type VoteCastsResponse struct {
	VoteCasts []VoteCast `json:"voteCasts"`
}

type ProposalVotersResponse struct {
	Proposals []struct {
		Voters []VoteCast `json:"voters"`
	} `json:"proposals"`
}

type Contributor struct {
	ID                      string  `json:"id"`
	Power                   string  `json:"power"`
	Balance                 *string `json:"balance"`
	DelegatesCountAll       int     `json:"delegatesCountAll"`
	DelegatesCountEffective int     `json:"delegatesCountEffective"`
}

type ContributorsResponse struct {
	Contributors []Contributor `json:"contributors"`
}

type ProposalScope struct {
	ChainID         int
	DaoCode         string
	GovernorAddress string
}

func (s ProposalScope) withScope(where map[string]any) map[string]any {
	if where == nil {
		where = map[string]any{}
	}
	if s.ChainID != 0 {
		where["chainId_eq"] = s.ChainID
	}
	if s.DaoCode != "" {
		where["daoCode_eq"] = s.DaoCode
	}
	if s.GovernorAddress != "" {
		where["governorAddress_eq"] = strings.ToLower(s.GovernorAddress)
	}
	return where
}

// DegovIndexer handles GraphQL queries to fetch governance data
type DegovIndexer struct {
	client   *graphql.Client
	endpoint string
	now      func() time.Time
}

// NewDegovIndexer creates a new DegovIndexer instance with the given endpoint
func NewDegovIndexer(endpoint string) *DegovIndexer {
	client := graphql.NewClient(endpoint)
	return &DegovIndexer{
		client:   client,
		endpoint: endpoint,
		now:      time.Now,
	}
}

// GetEndpoint returns the current GraphQL endpoint
func (d *DegovIndexer) GetEndpoint() string {
	return d.endpoint
}

// QueryDataMetrics executes the QueryDataMetrics GraphQL query and returns a single DataMetrics object
func (d *DegovIndexer) QueryGlobalDataMetrics(scope ProposalScope) (*DataMetrics, error) {
	query := `
		query QueryDataMetrics($where: DataMetricWhereInput) {
			dataMetrics(where: $where) {
				proposalsCount
				memberCount
				holdersCount
				contributorCount
				powerSum
				votesCount
				votesWeightAbstainSum
				votesWeightAgainstSum
				votesWeightForSum
				votesWithParamsCount
				votesWithoutParamsCount
				id
			}
		}
	`

	req := graphql.NewRequest(query)
	req.Var("where", scope.withScope(map[string]any{
		"id_eq": "global",
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var response DataMetricsResponse
	if err := d.client.Run(ctx, req, &response); err != nil {
		return nil, fmt.Errorf("failed to execute QueryDataMetrics: %w", err)
	}

	metrics := DataMetrics{}
	hasGlobalMetrics := len(response.DataMetrics) > 0
	if hasGlobalMetrics {
		metrics = response.DataMetrics[0]
	}

	if !hasGlobalMetrics || metrics.ProposalsCount == nil {
		proposalsCount, err := d.QueryProposalsCount(scope)
		if err != nil {
			if hasGlobalMetrics {
				return &metrics, nil
			}
			return nil, err
		}

		metrics.ProposalsCount = &proposalsCount
	}

	return &metrics, nil
}

func (d *DegovIndexer) QueryProposalsCount(scope ProposalScope) (int, error) {
	query := `
		query QueryProposalsCount($where: ProposalWhereInput, $limit: Int!, $offset: Int!) {
			proposalsPage(where: $where, limit: $limit, offset: $offset) {
				totalCount
				items {
					id
				}
			}
		}
	`

	req := graphql.NewRequest(query)
	req.Var("where", scope.withScope(nil))
	req.Var("limit", 1)
	req.Var("offset", 0)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var response ProposalPageResponse
	if err := d.client.Run(ctx, req, &response); err != nil {
		return 0, fmt.Errorf("failed to execute QueryProposalsCount: %w", err)
	}

	return response.ProposalsPage.TotalCount, nil
}

func (d *DegovIndexer) InspectProposal(scope ProposalScope, proposalId string) (*Proposal, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return d.InspectProposalWithContext(ctx, scope, proposalId)
}

func (d *DegovIndexer) InspectProposalWithContext(ctx context.Context, scope ProposalScope, proposalId string) (*Proposal, error) {
	query := `
		query QueryProposal($where: ProposalWhereInput!) {
			proposals(where: $where) {
				id
				chainId
				daoCode
				governorAddress
				proposalId
				title
				quorum
				voteStartTimestamp
				voteEndTimestamp
				voteStart
				voteEnd
				decimals
				blockInterval
				clockMode
				proposer
				blockNumber
				blockTimestamp
				transactionHash
				proposalDeadline
				proposalEta
				queueReadyAt
				queueExpiresAt
				timelockAddress
				timelockGracePeriod
				description

				metricsVotesCount
				metricsVotesWeightAbstainSum
				metricsVotesWeightAgainstSum
				metricsVotesWeightForSum
				metricsVotesWithParamsCount
				metricsVotesWithoutParamsCount
			}
		}
	`

	req := graphql.NewRequest(query)
	req.Var("where", scope.withScope(map[string]any{
		"proposalId_eq": proposalId,
	}))

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var response ProposalsResponse
	if err := d.client.Run(ctx, req, &response); err != nil {
		return nil, fmt.Errorf("failed to execute QueryProposal: %w", err)
	}

	// Return the first item if available, otherwise return nil
	if len(response.Proposals) > 0 {
		return &response.Proposals[0], nil
	}

	return nil, fmt.Errorf("no proposal found with id %s", proposalId)
}

// QueryProposalsByBlockNumber queries proposals after the given blockNumber/id cursor.
func (d *DegovIndexer) QueryProposalsByBlockNumber(scope ProposalScope, afterBlockNumber int64, afterProposalID string) ([]Proposal, error) {
	const limit = 30
	query := `
		query QueryProposalsByBlockNumber($limit: Int!, $where: ProposalWhereInput) {
			proposals(orderBy: [blockNumber_ASC_NULLS_FIRST, id_ASC], limit: $limit, where: $where) {
				id
				chainId
				daoCode
				governorAddress
				proposalId
				title
				quorum
				voteStartTimestamp
				voteEndTimestamp
				voteStart
				voteEnd
				decimals
				blockInterval
				clockMode
				proposer
				blockNumber
				blockTimestamp
				transactionHash
				proposalDeadline
				proposalEta
				queueReadyAt
				queueExpiresAt
				timelockAddress
				timelockGracePeriod
				description
				metricsVotesCount
				metricsVotesWeightAbstainSum
				metricsVotesWeightAgainstSum
				metricsVotesWeightForSum
				metricsVotesWithParamsCount
				metricsVotesWithoutParamsCount
			}
		}
	`

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := graphql.NewRequest(query)
	req.Var("limit", limit)
	req.Var("where", scope.withScope(map[string]any{
		"OR": []map[string]any{
			{"blockNumber_gt": strconv.FormatInt(afterBlockNumber, 10)},
			{"blockNumber_eq": strconv.FormatInt(afterBlockNumber, 10), "id_gt": afterProposalID},
		},
	}))

	var response ProposalsResponse
	if err := d.client.Run(ctx, req, &response); err != nil {
		return nil, fmt.Errorf("failed to execute QueryProposalsByBlockNumber: %w", err)
	}
	if len(response.Proposals) > limit {
		return nil, fmt.Errorf("QueryProposalsByBlockNumber returned %d proposals, limit is %d", len(response.Proposals), limit)
	}

	previousBlockNumber := afterBlockNumber
	previousProposalID := afterProposalID
	for _, proposal := range response.Proposals {
		blockNumber, err := strconv.ParseInt(proposal.BlockNumber, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid blockNumber %q for proposal %q: %w", proposal.BlockNumber, proposal.ID, err)
		}
		if blockNumber < previousBlockNumber || (blockNumber == previousBlockNumber && proposal.ID <= previousProposalID) {
			return nil, fmt.Errorf("proposal cursor regressed at blockNumber %d id %q", blockNumber, proposal.ID)
		}
		previousBlockNumber = blockNumber
		previousProposalID = proposal.ID
	}

	return response.Proposals, nil
}

func (d *DegovIndexer) QueryVotesOffset(ctx context.Context, scope ProposalScope, offset int, proposalId string) ([]VoteCast, error) {
	return d.QueryVotes(ctx, scope, offset, 30, proposalId)
}

func (d *DegovIndexer) QueryVotes(ctx context.Context, scope ProposalScope, offset int, limit int, proposalId string) ([]VoteCast, error) {
	query := `
		query QueryVotesOffset($limit: Int!, $offset: Int!, $where: ProposalWhereInput!) {
			proposals(orderBy: [id_ASC], limit: 2, where: $where) {
				voters(orderBy: [blockTimestamp_ASC_NULLS_LAST, id_ASC], limit: $limit, offset: $offset) {
					reason
					support
					voter
					weight
					transactionHash
					id
					blockNumber
					blockTimestamp
				}
			}
		}
	`
	req := graphql.NewRequest(query)
	req.Var("limit", limit)
	req.Var("offset", offset)
	req.Var("where", scope.withScope(map[string]any{
		"proposalId_eq": proposalId,
	}))

	var response ProposalVotersResponse
	if err := d.client.Run(ctx, req, &response); err != nil {
		return nil, fmt.Errorf("failed to execute QueryVotesOffset: %w", err)
	}
	if len(response.Proposals) > 1 {
		return nil, fmt.Errorf("multiple proposals found for proposalId %s", proposalId)
	}
	if len(response.Proposals) == 0 {
		return nil, nil
	}

	votes := response.Proposals[0].Voters
	for i := range votes {
		votes[i].ProposalID = proposalId
	}
	return votes, nil
}

func (d *DegovIndexer) QueryContributors(ctx context.Context, scope ProposalScope, offset int, limit int, orderBy string) ([]Contributor, error) {
	switch orderBy {
	case "power_ASC", "id_ASC":
	default:
		orderBy = "power_DESC"
	}

	query := fmt.Sprintf(`
		query QueryContributors($limit: Int!, $offset: Int!, $where: ContributorWhereInput) {
			contributors(orderBy: %s, limit: $limit, offset: $offset, where: $where) {
				id
				power
				balance
				delegatesCountAll
				delegatesCountEffective
			}
		}
	`, orderBy)
	req := graphql.NewRequest(query)
	req.Var("limit", limit)
	req.Var("offset", offset)
	req.Var("where", scope.withScope(nil))

	var response ContributorsResponse
	if err := d.client.Run(ctx, req, &response); err != nil {
		return nil, fmt.Errorf("failed to execute QueryContributors: %w", err)
	}

	return response.Contributors, nil
}

func (d *DegovIndexer) QueryContributor(ctx context.Context, scope ProposalScope, address string) (*Contributor, error) {
	query := `
		query QueryContributor($where: ContributorWhereInput!) {
			contributors(where: $where) {
				id
				power
				balance
				delegatesCountAll
				delegatesCountEffective
			}
		}
	`
	req := graphql.NewRequest(query)
	req.Var("where", scope.withScope(map[string]any{
		"id_eq": strings.ToLower(address),
	}))

	var response ContributorsResponse
	if err := d.client.Run(ctx, req, &response); err != nil {
		return nil, fmt.Errorf("failed to execute QueryContributor: %w", err)
	}
	if len(response.Contributors) > 0 {
		return &response.Contributors[0], nil
	}

	return nil, fmt.Errorf("no contributor found with address %s", address)
}

func (d *DegovIndexer) QueryVote(scope ProposalScope, proposalId string, id string) (*VoteCast, error) {
	query := `
		query QueryVote($where: ProposalWhereInput!, $voterWhere: VoteCastGroupWhereInput!) {
			proposals(orderBy: [id_ASC], limit: 2, where: $where) {
				voters(where: $voterWhere, orderBy: [id_ASC], limit: 2) {
					reason
					support
					voter
					weight
					transactionHash
					id
					blockNumber
					blockTimestamp
				}
			}
		}
	`

	req := graphql.NewRequest(query)
	req.Var("where", scope.withScope(map[string]any{
		"proposalId_eq": proposalId,
	}))
	req.Var("voterWhere", map[string]any{"id_eq": id})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var response ProposalVotersResponse
	if err := d.client.Run(ctx, req, &response); err != nil {
		return nil, fmt.Errorf("failed to execute QueryVote: %w", err)
	}
	if len(response.Proposals) > 1 {
		return nil, fmt.Errorf("multiple proposals found for proposalId %s", proposalId)
	}
	if len(response.Proposals) == 1 {
		votes := response.Proposals[0].Voters
		if len(votes) > 1 {
			return nil, fmt.Errorf("multiple votes found with id %s", id)
		}
		if len(votes) == 1 {
			vote := votes[0]
			vote.ProposalID = proposalId
			return &vote, nil
		}
	}

	return nil, fmt.Errorf("no vote found with id %s", id)
}

func (d *DegovIndexer) QueryVoteByVoter(scope ProposalScope, proposalId string, voter string) (*VoteCast, error) {
	query := `
		query QueryVoteByVoter($where: ProposalWhereInput!, $voterWhere: VoteCastGroupWhereInput!) {
			proposals(orderBy: [id_ASC], limit: 2, where: $where) {
				voters(where: $voterWhere, orderBy: [blockTimestamp_ASC_NULLS_LAST, id_ASC], limit: 1) {
					reason
					support
					voter
					weight
					transactionHash
					id
					blockNumber
					blockTimestamp
				}
			}
		}
	`

	req := graphql.NewRequest(query)
	req.Var("where", scope.withScope(map[string]any{
		"proposalId_eq": proposalId,
	}))
	req.Var("voterWhere", map[string]any{"voter_eq": voter})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var response ProposalVotersResponse
	if err := d.client.Run(ctx, req, &response); err != nil {
		return nil, fmt.Errorf("failed to execute QueryVoteByVoter: %w", err)
	}
	if len(response.Proposals) > 1 {
		return nil, fmt.Errorf("multiple proposals found for proposalId %s", proposalId)
	}
	if len(response.Proposals) == 1 && len(response.Proposals[0].Voters) > 0 {
		vote := response.Proposals[0].Voters[0]
		vote.ProposalID = proposalId
		return &vote, nil
	}

	return nil, fmt.Errorf("no vote found for proposalId %s and voter %s", proposalId, voter)
}

func (d *DegovIndexer) QueryExpiringProposals(scope ProposalScope) ([]Proposal, error) {
	query := `
	query QueryExpiringProposals($limit: Int!, $offset: Int!, $where: ProposalWhereInput!) {
	  proposals(
	    limit: $limit
	    offset: $offset
	    orderBy: [blockTimestamp_ASC_NULLS_FIRST, id_ASC]
	    where: $where
	  ) {
	    id
	    chainId
	    daoCode
	    governorAddress
	    proposalId
	    title
	    quorum
	    voteStartTimestamp
	    voteEndTimestamp
	    voteStart
	    voteEnd
	    decimals
	    blockInterval
	    clockMode
	    proposer
	    blockNumber
	    blockTimestamp
	    transactionHash
	    proposalDeadline
	    proposalEta
	    queueReadyAt
	    queueExpiresAt
	    timelockAddress
	    timelockGracePeriod
	    metricsVotesCount
	    metricsVotesWeightAbstainSum
	    metricsVotesWeightAgainstSum
	    metricsVotesWeightForSum
	    description
	  }
	}
	`

	const limit = 50
	var proposals []Proposal

	now := d.now()
	startTimestamp := now.UnixMilli()
	endTimestamp := now.Add(48 * time.Hour).UnixMilli()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	seenIDs := make(map[string]struct{})
	for offset := 0; ; offset += limit {
		req := graphql.NewRequest(query)

		req.Var("limit", limit)
		req.Var("offset", offset)
		req.Var("where", scope.withScope(map[string]any{
			"voteEndTimestamp_gte": strconv.FormatInt(startTimestamp, 10),
			"voteEndTimestamp_lt":  strconv.FormatInt(endTimestamp, 10),
		}))

		var response ProposalsResponse

		if err := d.client.Run(ctx, req, &response); err != nil {
			return nil, fmt.Errorf("graphql query failed on offset %d: %w", offset, err)
		}
		newIDs := 0
		for _, proposal := range response.Proposals {
			if _, exists := seenIDs[proposal.ID]; exists {
				continue
			}
			seenIDs[proposal.ID] = struct{}{}
			newIDs++
			proposals = append(proposals, proposal)
		}
		if len(response.Proposals) < limit {
			break
		}
		if newIDs == 0 {
			return nil, fmt.Errorf("expiring proposal pagination made no progress at offset %d", offset)
		}
	}

	return proposals, nil
}

// Delegate represents a delegation record
type Delegate struct {
	ID           string `json:"id"`
	Power        string `json:"power"`
	FromDelegate string `json:"fromDelegate"`
	ToDelegate   string `json:"toDelegate"`
}

// DelegatesResponse represents the GraphQL response for delegates query
type DelegatesResponse struct {
	Delegates []Delegate `json:"delegates"`
}

// QueryDelegatorsTo queries all delegators who delegated to the given address (excluding self-delegation)
// Returns true if there are delegators other than the address itself
func (d *DegovIndexer) QueryDelegatorsTo(ctx context.Context, scope ProposalScope, toAddress string) ([]Delegate, error) {
	query := `
		query QueryDelegates($where: DelegateWhereInput) {
			delegates(where: $where) {
				id
				power
				fromDelegate
				toDelegate
			}
		}
	`

	req := graphql.NewRequest(query)
	req.Var("where", scope.withScope(map[string]any{
		"toDelegate_eq":       toAddress,
		"fromDelegate_not_eq": toAddress,
	}))

	var response DelegatesResponse
	if err := d.client.Run(ctx, req, &response); err != nil {
		return nil, fmt.Errorf("failed to query delegates: %w", err)
	}

	return response.Delegates, nil
}

// HasDelegatorsOtherThanSelf checks if there are any delegators to the given address (excluding self)
func (d *DegovIndexer) HasDelegatorsOtherThanSelf(ctx context.Context, scope ProposalScope, toAddress string) (bool, error) {
	delegates, err := d.QueryDelegatorsTo(ctx, scope, toAddress)
	if err != nil {
		return false, err
	}
	return len(delegates) > 0, nil
}
