package internal

import (
	"context"
	"fmt"

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
func (d *DegovIndexer) QueryGlobalDataMetrics(ctx context.Context) (*DataMetrics, error) {
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
