package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/machinebox/graphql"
)

// DataMetrics represents the data metrics structure from GraphQL response
type DataMetrics struct {
	ProposalsCount          int    `json:"proposalsCount"`
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

// Proposal represents the structure of a governance proposal.
type Proposal struct {
	ID                           string  `json:"id"`
	ProposalID                   string  `json:"proposalId"`
	Title                        string  `json:"title"`
	Quorum                       string  `json:"quorum"`
	VoteStartTimestamp           string  `json:"voteStartTimestamp"`
	VoteEndTimestamp             string  `json:"voteEndTimestamp"`
	VoteStart                    string  `json:"voteStart"`
	VoteEnd                      string  `json:"voteEnd"`
	Decimals                     string  `json:"decimals"`
	BlockInterval                string  `json:"blockInterval"`
	ClockMode                    string  `json:"clockMode"`
	Proposer                     string  `json:"proposer"`
	BlockNumber                  string  `json:"blockNumber"`
	BlockTimestamp               string  `json:"blockTimestamp"`
	TransactionHash              string  `json:"transactionHash"`
	MetricsVotesCount            *int    `json:"metricsVotesCount"`
	MetricsVotesWeightAbstainSum *string `json:"metricsVotesWeightAbstainSum"`
	MetricsVotesWeightAgainstSum *string `json:"metricsVotesWeightAgainstSum"`
	MetricsVotesWeightForSum     *string `json:"metricsVotesWeightForSum"`
	Description                  string  `json:"description"`
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
func (d *DegovIndexer) QueryGlobalDataMetrics() (*DataMetrics, error) {
	query := `
		query QueryDataMetrics {
			dataMetrics(where: {id_eq: "global"}) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var response DataMetricsResponse
	if err := d.client.Run(ctx, req, &response); err != nil {
		return nil, fmt.Errorf("failed to execute QueryDataMetrics: %w", err)
	}

	// Return the first item if available, otherwise return nil
	if len(response.DataMetrics) > 0 {
		return &response.DataMetrics[0], nil
	}

	return nil, fmt.Errorf("no data metrics found for global id")
}

func (d *DegovIndexer) InspectProposal(proposalId string) (*Proposal, error) {
	query := `
		query QueryProposal($proposalId: String!) {
			proposals(where: {proposalId_eq: $proposalId}) {
				id
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
	req.Var("proposalId", proposalId)

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
func (d *DegovIndexer) QueryProposalsOffset(offset int) ([]Proposal, error) {
	query := `
		query QueryProposalsOffset($limit: Int!, $offset: Int!) {
			proposals(orderBy: blockNumber_ASC_NULLS_FIRST, limit: $limit, offset: $offset) {
				id
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var response ProposalsResponse
	if err := d.client.Run(ctx, req, &response); err != nil {
		return nil, fmt.Errorf("failed to execute QueryProposalsOffset: %w", err)
	}

	return response.Proposals, nil
}

func (d *DegovIndexer) QueryVotesOffset(ctx context.Context, offset int, proposalId string) ([]VoteCast, error) {
	query := `
		query QueryVotesOffset($limit: Int!, $offset: Int!, $proposalId: String!) {
			voteCasts(orderBy: blockNumber_ASC_NULLS_FIRST, limit: $limit, offset: $offset, where: {proposalId_eq: $proposalId}) {
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
	req.Var("proposalId", proposalId)

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

func (d *DegovIndexer) QueryVoteByVoter(proposalId string, voter string) (*VoteCast, error) {
	query := `
		query QueryVoteByVoter($proposalId: String!, $voter: String!) {
			voteCasts(where: {proposalId_eq: $proposalId, voter_eq: $voter}) {
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
	req.Var("proposalId", proposalId)
	req.Var("voter", voter)

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

func (d *DegovIndexer) QueryExpiringProposals() ([]Proposal, error) {
	query := `
	query QueryExpiringProposals($limit: Int!, $offset: Int!, $start: BigInt!, $end: BigInt!) {
	  proposals(
	    limit: $limit
	    offset: $offset
	    orderBy: blockTimestamp_ASC_NULLS_FIRST
	    where: {
	      voteEndTimestamp_gte: $start
	      voteEndTimestamp_lt: $end
	    }
	  ) {
	    id
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
		req.Var("start", startTimestamp)
		req.Var("end", endTimestamp)

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
