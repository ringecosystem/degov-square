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
	MemberCount             int    `json:"memberCount"`
	PowerSum                string `json:"powerSum"`
	VotesCount              int    `json:"votesCount"`
	VotesWeightAbstainSum   string `json:"votesWeightAbstainSum"`
	VotesWeightAgainstSum   string `json:"votesWeightAgainstSum"`
	VotesWeightForSum       string `json:"votesWeightForSum"`
	VotesWithParamsCount    int    `json:"votesWithParamsCount"`
	VotesWithoutParamsCount int    `json:"votesWithoutParamsCount"`
	ID                      string `json:"id"`
}

// DataMetricsResponse represents the GraphQL response structure
type DataMetricsResponse struct {
	DataMetrics []DataMetrics `json:"dataMetrics"`
}

type ProposalConnection struct {
	TotalCount int `json:"totalCount"`
}

type ProposalConnectionResponse struct {
	ProposalsConnection ProposalConnection `json:"proposalsConnection"`
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
}

// NewDegovIndexer creates a new DegovIndexer instance with the given endpoint
func NewDegovIndexer(endpoint string) *DegovIndexer {
	client := graphql.NewClient(endpoint)
	return &DegovIndexer{
		client:   client,
		endpoint: endpoint,
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
		query QueryProposalsCount($where: ProposalWhereInput) {
			proposalsConnection(orderBy: [id_ASC], where: $where) {
				totalCount
			}
		}
	`

	req := graphql.NewRequest(query)
	req.Var("where", scope.withScope(nil))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var response ProposalConnectionResponse
	if err := d.client.Run(ctx, req, &response); err != nil {
		return 0, fmt.Errorf("failed to execute QueryProposalsCount: %w", err)
	}

	return response.ProposalsConnection.TotalCount, nil
}

func (d *DegovIndexer) InspectProposal(scope ProposalScope, proposalId string) (*Proposal, error) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

// QueryProposalsOffset executes the QueryProposalsOffset GraphQL query and returns proposals list
func (d *DegovIndexer) QueryProposalsOffset(scope ProposalScope, offset int) ([]Proposal, error) {
	query := `
		query QueryProposalsOffset($limit: Int!, $offset: Int!, $where: ProposalWhereInput) {
			proposals(orderBy: blockNumber_ASC_NULLS_FIRST, limit: $limit, offset: $offset, where: $where) {
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
	req.Var("limit", 30)
	req.Var("offset", offset)
	req.Var("where", scope.withScope(nil))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var response ProposalsResponse
	if err := d.client.Run(ctx, req, &response); err != nil {
		return nil, fmt.Errorf("failed to execute QueryProposalsOffset: %w", err)
	}

	return response.Proposals, nil
}

// QueryProposalsByBlockNumber queries proposals with blockNumber greater than the given value
func (d *DegovIndexer) QueryProposalsByBlockNumber(scope ProposalScope, afterBlockNumber int64) ([]Proposal, error) {
	query := `
		query QueryProposalsByBlockNumber($limit: Int!, $where: ProposalWhereInput) {
			proposals(orderBy: blockNumber_ASC_NULLS_FIRST, limit: $limit, where: $where) {
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
	blockNumberStr := strconv.FormatInt(afterBlockNumber, 10)
	whereFilter := map[string]any{"blockNumber_gt": blockNumberStr}

	req := graphql.NewRequest(query)
	req.Var("limit", 30)
	req.Var("where", scope.withScope(whereFilter))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var response ProposalsResponse
	if err := d.client.Run(ctx, req, &response); err != nil {
		return nil, fmt.Errorf("failed to execute QueryProposalsByBlockNumber: %w", err)
	}

	return response.Proposals, nil
}

func (d *DegovIndexer) QueryVotesOffset(ctx context.Context, scope ProposalScope, offset int, proposalId string) ([]VoteCast, error) {
	query := `
		query QueryVotesOffset($limit: Int!, $offset: Int!, $where: VoteCastWhereInput!) {
			voteCasts(orderBy: blockNumber_ASC_NULLS_FIRST, limit: $limit, offset: $offset, where: $where) {
				proposalId
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
	`
	req := graphql.NewRequest(query)
	req.Var("limit", 30)
	req.Var("offset", offset)
	req.Var("where", scope.withScope(map[string]any{
		"proposalId_eq": proposalId,
	}))

	var response VoteCastsResponse
	if err := d.client.Run(ctx, req, &response); err != nil {
		return nil, fmt.Errorf("failed to execute QueryVotesOffset: %w", err)
	}

	return response.VoteCasts, nil
}

func (d *DegovIndexer) QueryVote(id string) (*VoteCast, error) {
	query := `
	query QueryVote($id: String!) {
		voteCasts(where: {id_eq: $id}) {
			proposalId
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
	`

	req := graphql.NewRequest(query)
	req.Var("id", id)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var response VoteCastsResponse
	if err := d.client.Run(ctx, req, &response); err != nil {
		return nil, fmt.Errorf("failed to execute QueryVotesOffset: %w", err)
	}
	voteCasts := response.VoteCasts
	if len(voteCasts) > 0 {
		return &voteCasts[0], nil
	}

	return nil, fmt.Errorf("no vote found with id %s", id)
}

func (d *DegovIndexer) QueryVoteByVoter(scope ProposalScope, proposalId string, voter string) (*VoteCast, error) {
	query := `
		query QueryVoteByVoter($where: VoteCastWhereInput!) {
			voteCasts(where: $where) {
				proposalId
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
	`

	req := graphql.NewRequest(query)
	req.Var("where", scope.withScope(map[string]any{
		"proposalId_eq": proposalId,
		"voter_eq":      voter,
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var response VoteCastsResponse
	if err := d.client.Run(ctx, req, &response); err != nil {
		return nil, fmt.Errorf("failed to execute QueryVoteByVoter: %w", err)
	}
	if len(response.VoteCasts) > 0 {
		return &response.VoteCasts[0], nil
	}

	return nil, fmt.Errorf("no vote found for proposalId %s and voter %s", proposalId, voter)
}

func (d *DegovIndexer) QueryExpiringProposals(scope ProposalScope) ([]Proposal, error) {
	query := `
	query QueryExpiringProposals($limit: Int!, $offset: Int!, $where: ProposalWhereInput!) {
	  proposals(
	    limit: $limit
	    offset: $offset
	    orderBy: blockTimestamp_ASC_NULLS_FIRST
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
	var offset = 0
	var allProposals []Proposal

	now := time.Now()
	startTimestamp := now.UnixMilli()
	endTimestamp := now.Add(2 * 24 * 60 * time.Minute).UnixMilli()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for {
		req := graphql.NewRequest(query)

		req.Var("limit", limit)
		req.Var("offset", offset)
		req.Var("where", scope.withScope(map[string]any{
			"voteEndTimestamp_gte": startTimestamp,
			"voteEndTimestamp_lt":  endTimestamp,
		}))

		var response ProposalsResponse

		if err := d.client.Run(ctx, req, &response); err != nil {
			return nil, fmt.Errorf("graphql query failed on offset %d: %w", offset, err)
		}
		if len(response.Proposals) == 0 {
			break
		}
		allProposals = append(allProposals, response.Proposals...)
		offset += len(response.Proposals)
	}

	return allProposals, nil
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
func (d *DegovIndexer) QueryDelegatorsTo(ctx context.Context, toAddress string) ([]Delegate, error) {
	query := `
		query QueryDelegates($toDelegate: String!, $fromDelegate: String!) {
			delegates(where: {toDelegate_eq: $toDelegate, fromDelegate_not_eq: $fromDelegate}) {
				id
				power
				fromDelegate
				toDelegate
			}
		}
	`

	req := graphql.NewRequest(query)
	req.Var("toDelegate", toAddress)
	req.Var("fromDelegate", toAddress)

	var response DelegatesResponse
	if err := d.client.Run(ctx, req, &response); err != nil {
		return nil, fmt.Errorf("failed to query delegates: %w", err)
	}

	return response.Delegates, nil
}

// HasDelegatorsOtherThanSelf checks if there are any delegators to the given address (excluding self)
func (d *DegovIndexer) HasDelegatorsOtherThanSelf(ctx context.Context, toAddress string) (bool, error) {
	delegates, err := d.QueryDelegatorsTo(ctx, toAddress)
	if err != nil {
		return false, err
	}
	return len(delegates) > 0, nil
}
