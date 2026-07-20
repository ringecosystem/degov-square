package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestQueryProposalsByBlockNumberPaginatesAndFiltersCursorDeterministically(t *testing.T) {
	type graphqlRequest struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}

	longID := strings.Repeat("x", 369)
	page := make([]Proposal, 30)
	for i := range page {
		page[i] = Proposal{ID: fmt.Sprintf("old-%02d", i), BlockNumber: "99"}
	}
	page[5] = Proposal{ID: "later-block", BlockNumber: "102"}

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		for _, unsupported := range []string{
			"blockNumber_ASC_NULLS_FIRST",
			"blockNumber_eq",
			"blockNumber_gt",
			"id_gt",
		} {
			if strings.Contains(req.Query, unsupported) {
				t.Fatalf("query contains unsupported schema value %q: %s", unsupported, req.Query)
			}
		}
		if !strings.Contains(req.Query, "orderBy: [blockTimestamp_DESC_NULLS_LAST, id_ASC]") {
			t.Fatalf("query = %s, want supported stable order", req.Query)
		}
		where, ok := req.Variables["where"].(map[string]any)
		if !ok {
			t.Fatalf("where = %#v, want map", req.Variables["where"])
		}
		if len(where) != 3 {
			t.Fatalf("where = %#v, want only scope fields", where)
		}
		if got, want := req.Variables["limit"], float64(30); got != want {
			t.Fatalf("limit = %#v, want %#v", got, want)
		}
		if got, want := req.Variables["offset"], float64(requestCount*30); got != want {
			t.Fatalf("offset = %#v, want %#v", got, want)
		}

		w.Header().Set("Content-Type", "application/json")
		switch requestCount {
		case 0:
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"proposals": page}})
		case 1:
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"proposals": []Proposal{
				{ID: longID, BlockNumber: "101"},
				{ID: "z", BlockNumber: "100"},
				{ID: "a", BlockNumber: "100"},
				{ID: "later-block", BlockNumber: "102"},
			}}})
		default:
			t.Fatalf("unexpected request %d", requestCount)
		}
		requestCount++
	}))
	defer server.Close()

	indexer := NewDegovIndexer(server.URL)
	proposals, err := indexer.QueryProposalsByBlockNumber(ProposalScope{
		ChainID:         46,
		DaoCode:         "ring-dao",
		GovernorAddress: "0xAbC123",
	}, 100, "m")
	if err != nil {
		t.Fatalf("QueryProposalsByBlockNumber() error = %v", err)
	}
	if got, want := requestCount, 2; got != want {
		t.Fatalf("request count = %d, want %d", got, want)
	}
	if got, want := len(proposals), 3; got != want {
		t.Fatalf("len(proposals) = %d, want %d: %#v", got, want, proposals)
	}
	if got, want := []string{proposals[0].ID, proposals[1].ID, proposals[2].ID}, []string{"z", longID, "later-block"}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("proposal ids = %#v, want %#v", got, want)
	}
}

func TestQueryExpiringProposalsPaginatesAndFiltersFortyEightHourWindow(t *testing.T) {
	type graphqlRequest struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}

	now := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)
	page := make([]Proposal, 50)
	for i := range page {
		page[i] = Proposal{ID: fmt.Sprintf("expired-%02d", i), VoteEndTimestamp: fmt.Sprint(now.Add(-time.Hour).UnixMilli())}
	}
	page[0] = Proposal{ID: "window-start", VoteEndTimestamp: fmt.Sprint(now.UnixMilli())}

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		for _, unsupported := range []string{
			"blockTimestamp_ASC_NULLS_FIRST",
			"voteEndTimestamp_gte",
			"voteEndTimestamp_lt",
		} {
			if strings.Contains(req.Query, unsupported) {
				t.Fatalf("query contains unsupported schema value %q: %s", unsupported, req.Query)
			}
		}
		if !strings.Contains(req.Query, "orderBy: [blockTimestamp_DESC_NULLS_LAST, id_ASC]") {
			t.Fatalf("query = %s, want supported stable order", req.Query)
		}
		where, ok := req.Variables["where"].(map[string]any)
		if !ok || len(where) != 3 {
			t.Fatalf("where = %#v, want only scope fields", req.Variables["where"])
		}
		if got, want := req.Variables["limit"], float64(50); got != want {
			t.Fatalf("limit = %#v, want %#v", got, want)
		}
		if got, want := req.Variables["offset"], float64(requestCount*50); got != want {
			t.Fatalf("offset = %#v, want %#v", got, want)
		}

		w.Header().Set("Content-Type", "application/json")
		switch requestCount {
		case 0:
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"proposals": page}})
		case 1:
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"proposals": []Proposal{
				{ID: "window-start", VoteEndTimestamp: fmt.Sprint(now.UnixMilli())},
				{ID: "window-end-minus-one", VoteEndTimestamp: fmt.Sprint(now.Add(48*time.Hour).UnixMilli() - 1)},
				{ID: "window-end", VoteEndTimestamp: fmt.Sprint(now.Add(48 * time.Hour).UnixMilli())},
				{ID: "before-window", VoteEndTimestamp: fmt.Sprint(now.UnixMilli() - 1)},
				{ID: "missing-time"},
			}}})
		default:
			t.Fatalf("unexpected request %d", requestCount)
		}
		requestCount++
	}))
	defer server.Close()

	indexer := NewDegovIndexer(server.URL)
	indexer.now = func() time.Time { return now }
	proposals, err := indexer.QueryExpiringProposals(ProposalScope{
		ChainID:         46,
		DaoCode:         "ring-dao",
		GovernorAddress: "0xAbC123",
	})
	if err != nil {
		t.Fatalf("QueryExpiringProposals() error = %v", err)
	}
	if got, want := requestCount, 2; got != want {
		t.Fatalf("request count = %d, want %d", got, want)
	}
	if got, want := []string{proposals[0].ID, proposals[1].ID}, []string{"window-start", "window-end-minus-one"}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("proposal ids = %#v, want %#v", got, want)
	}
}

func TestQueryVotesUsesNestedProposalVotersWithPagination(t *testing.T) {
	type graphqlRequest struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}

	scope := ProposalScope{
		ChainID:         46,
		DaoCode:         "ring-dao",
		GovernorAddress: "0xAbC123",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if strings.Contains(req.Query, "voteCasts") || strings.Contains(req.Query, "VoteCastWhereInput") {
			t.Fatalf("query uses removed top-level voteCasts schema: %s", req.Query)
		}
		if !strings.Contains(req.Query, "proposals(where: $where)") {
			t.Fatalf("query = %s, want scoped proposals query", req.Query)
		}
		if !strings.Contains(req.Query, "voters(orderBy: [blockTimestamp_ASC_NULLS_LAST, id_ASC], limit: $limit, offset: $offset)") {
			t.Fatalf("query = %s, want nested voters with stable pagination", req.Query)
		}
		if strings.Contains(req.Query, "\n\t\t\t\t\tproposalId\n") {
			t.Fatalf("query requests proposalId unavailable on nested voters: %s", req.Query)
		}
		if got, want := req.Variables["limit"], float64(7); got != want {
			t.Fatalf("limit = %#v, want %#v", got, want)
		}
		if got, want := req.Variables["offset"], float64(14); got != want {
			t.Fatalf("offset = %#v, want %#v", got, want)
		}
		where, ok := req.Variables["where"].(map[string]any)
		if !ok {
			t.Fatalf("where = %#v, want map", req.Variables["where"])
		}
		if got, want := where["proposalId_eq"], "proposal-1"; got != want {
			t.Fatalf("proposalId_eq = %#v, want %#v", got, want)
		}
		if got, want := where["chainId_eq"], float64(scope.ChainID); got != want {
			t.Fatalf("chainId_eq = %#v, want %#v", got, want)
		}
		if got, want := where["daoCode_eq"], scope.DaoCode; got != want {
			t.Fatalf("daoCode_eq = %#v, want %#v", got, want)
		}
		if got, want := where["governorAddress_eq"], "0xabc123"; got != want {
			t.Fatalf("governorAddress_eq = %#v, want %#v", got, want)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"proposals":[{"voters":[{"reason":"because","support":1,"voter":"0x1","weight":"10","transactionHash":"0xtx","id":"vote-1","blockNumber":"12","blockTimestamp":"1000"}]}]}}`))
	}))
	defer server.Close()

	indexer := NewDegovIndexer(server.URL)
	votes, err := indexer.QueryVotes(context.Background(), scope, 14, 7, "proposal-1")
	if err != nil {
		t.Fatalf("QueryVotes() error = %v", err)
	}
	if got, want := len(votes), 1; got != want {
		t.Fatalf("len(votes) = %d, want %d", got, want)
	}
	if got, want := votes[0].ID, "vote-1"; got != want {
		t.Fatalf("vote id = %q, want %q", got, want)
	}
	if got, want := votes[0].ProposalID, "proposal-1"; got != want {
		t.Fatalf("vote proposal id = %q, want %q", got, want)
	}
}

func TestQueryGlobalDataMetricsFallsBackToProposalsPageWhenGlobalProposalCountUnavailable(t *testing.T) {
	t.Parallel()

	type graphqlRequest struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}

	tests := []struct {
		name                string
		dataMetricsResponse string
	}{
		{
			name:                "null proposals count",
			dataMetricsResponse: `{"data":{"dataMetrics":[{"proposalsCount":null,"memberCount":5,"powerSum":"8","votesCount":3,"votesWeightAbstainSum":"0","votesWeightAgainstSum":"0","votesWeightForSum":"8","votesWithParamsCount":0,"votesWithoutParamsCount":3,"id":"global"}]}}`,
		},
		{
			name:                "missing global row",
			dataMetricsResponse: `{"data":{"dataMetrics":[]}}`,
		},
	}

	scope := ProposalScope{
		ChainID:         46,
		DaoCode:         "ring-dao",
		GovernorAddress: "0xAbC123",
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req graphqlRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Fatalf("decode request: %v", err)
				}

				where, ok := req.Variables["where"].(map[string]any)
				if !ok {
					t.Fatalf("expected where variables, got %#v", req.Variables)
				}
				if got, want := where["chainId_eq"], float64(scope.ChainID); got != want {
					t.Fatalf("chainId_eq = %#v, want %#v", got, want)
				}
				if got, want := where["daoCode_eq"], scope.DaoCode; got != want {
					t.Fatalf("daoCode_eq = %#v, want %#v", got, want)
				}
				if got, want := where["governorAddress_eq"], "0xabc123"; got != want {
					t.Fatalf("governorAddress_eq = %#v, want %#v", got, want)
				}

				w.Header().Set("Content-Type", "application/json")
				switch requestCount {
				case 0:
					if got, want := where["id_eq"], "global"; got != want {
						t.Fatalf("id_eq = %#v, want %#v", got, want)
					}
					_, _ = w.Write([]byte(tt.dataMetricsResponse))
				case 1:
					if !strings.Contains(req.Query, "proposalsPage") {
						t.Fatalf("query = %s, want proposalsPage", req.Query)
					}
					if strings.Contains(req.Query, "proposalsConnection") {
						t.Fatalf("query = %s, want no proposalsConnection", req.Query)
					}
					if _, exists := where["id_eq"]; exists {
						t.Fatalf("unexpected id_eq in proposal count fallback where: %#v", where)
					}
					if got, want := req.Variables["limit"], float64(1); got != want {
						t.Fatalf("limit = %#v, want %#v", got, want)
					}
					if got, want := req.Variables["offset"], float64(0); got != want {
						t.Fatalf("offset = %#v, want %#v", got, want)
					}
					_, _ = w.Write([]byte(`{"data":{"proposalsPage":{"totalCount":10,"items":[]}}}`))
				default:
					t.Fatalf("unexpected request count %d", requestCount)
				}
				requestCount++
			}))
			defer server.Close()

			indexer := NewDegovIndexer(server.URL)
			metrics, err := indexer.QueryGlobalDataMetrics(scope)
			if err != nil {
				t.Fatalf("QueryGlobalDataMetrics() error = %v", err)
			}
			if metrics == nil || metrics.ProposalsCount == nil {
				t.Fatalf("expected proposal count fallback, got %#v", metrics)
			}
			if got, want := *metrics.ProposalsCount, 10; got != want {
				t.Fatalf("proposal count = %d, want %d", got, want)
			}
			if got, want := requestCount, 2; got != want {
				t.Fatalf("request count = %d, want %d", got, want)
			}
		})
	}
}

func TestQueryGlobalDataMetricsPrefersHoldersCountForMembers(t *testing.T) {
	type graphqlRequest struct {
		Query string `json:"query"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(req.Query, "holdersCount") {
			t.Fatalf("query = %s, want holdersCount field", req.Query)
		}
		if !strings.Contains(req.Query, "contributorCount") {
			t.Fatalf("query = %s, want contributorCount field", req.Query)
		}
		if !strings.Contains(req.Query, "memberCount") {
			t.Fatalf("query = %s, want memberCount field", req.Query)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"dataMetrics":[{"proposalsCount":7,"memberCount":5,"holdersCount":6,"contributorCount":4,"powerSum":"8","votesCount":3,"votesWeightAbstainSum":"0","votesWeightAgainstSum":"0","votesWeightForSum":"8","votesWithParamsCount":0,"votesWithoutParamsCount":3,"id":"global"}]}}`))
	}))
	defer server.Close()

	indexer := NewDegovIndexer(server.URL)
	metrics, err := indexer.QueryGlobalDataMetrics(ProposalScope{DaoCode: "ring-dao"})
	if err != nil {
		t.Fatalf("QueryGlobalDataMetrics() error = %v", err)
	}
	if got, want := metrics.MemberCountValue(), 6; got == nil || *got != want {
		t.Fatalf("member count = %#v, want %d", got, want)
	}
	if metrics.ContributorCount == nil || *metrics.ContributorCount != 4 {
		t.Fatalf("contributor count = %#v, want 4", metrics.ContributorCount)
	}
}

func TestQueryGlobalDataMetricsFallsBackToMemberCountForMembers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"dataMetrics":[{"proposalsCount":7,"memberCount":5,"powerSum":"8","votesCount":3,"votesWeightAbstainSum":"0","votesWeightAgainstSum":"0","votesWeightForSum":"8","votesWithParamsCount":0,"votesWithoutParamsCount":3,"id":"global"}]}}`))
	}))
	defer server.Close()

	indexer := NewDegovIndexer(server.URL)
	metrics, err := indexer.QueryGlobalDataMetrics(ProposalScope{DaoCode: "ring-dao"})
	if err != nil {
		t.Fatalf("QueryGlobalDataMetrics() error = %v", err)
	}
	if got, want := metrics.MemberCountValue(), 5; got == nil || *got != want {
		t.Fatalf("member count = %#v, want %d", got, want)
	}
}

func TestQueryDelegatorsToScopesRequest(t *testing.T) {
	type graphqlRequest struct {
		Variables map[string]any `json:"variables"`
	}

	scope := ProposalScope{
		ChainID:         46,
		DaoCode:         "ring-dao",
		GovernorAddress: "0xAbC123",
	}
	toAddress := "0xDelegate"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		where, ok := req.Variables["where"].(map[string]any)
		if !ok {
			t.Fatalf("expected where variables, got %#v", req.Variables)
		}
		if got, want := where["toDelegate_eq"], toAddress; got != want {
			t.Fatalf("toDelegate_eq = %#v, want %#v", got, want)
		}
		if got, want := where["fromDelegate_not_eq"], toAddress; got != want {
			t.Fatalf("fromDelegate_not_eq = %#v, want %#v", got, want)
		}
		if got, want := where["chainId_eq"], float64(scope.ChainID); got != want {
			t.Fatalf("chainId_eq = %#v, want %#v", got, want)
		}
		if got, want := where["daoCode_eq"], scope.DaoCode; got != want {
			t.Fatalf("daoCode_eq = %#v, want %#v", got, want)
		}
		if got, want := where["governorAddress_eq"], "0xabc123"; got != want {
			t.Fatalf("governorAddress_eq = %#v, want %#v", got, want)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"delegates":[{"id":"1","power":"2","fromDelegate":"0xFrom","toDelegate":"0xDelegate"}]}}`))
	}))
	defer server.Close()

	indexer := NewDegovIndexer(server.URL)
	delegates, err := indexer.QueryDelegatorsTo(context.Background(), scope, toAddress)
	if err != nil {
		t.Fatalf("QueryDelegatorsTo() error = %v", err)
	}
	if len(delegates) != 1 {
		t.Fatalf("len(delegates) = %d, want 1", len(delegates))
	}
}

func TestInspectProposalWithContextCancelsRequest(t *testing.T) {
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"proposals":[]}}`))
	}))
	defer server.Close()

	indexer := NewDegovIndexer(server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := indexer.InspectProposalWithContext(ctx, ProposalScope{
		ChainID:         46,
		DaoCode:         "ring-dao",
		GovernorAddress: "0xAbC123",
	}, "0xcancel")
	if err == nil {
		t.Fatal("InspectProposalWithContext() error = nil, want cancellation error")
	}
	if got := requestCount.Load(); got != 0 {
		t.Fatalf("request count = %d, want 0", got)
	}
}

func TestQueryContributorsRequestsBalance(t *testing.T) {
	type graphqlRequest struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(req.Query, "balance") {
			t.Fatalf("query = %s, want balance field", req.Query)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"contributors":[{"id":"0x0000000000000000000000000000000000000001","power":"100","balance":"25","delegatesCountAll":2,"delegatesCountEffective":1}]}}`))
	}))
	defer server.Close()

	indexer := NewDegovIndexer(server.URL)
	contributors, err := indexer.QueryContributors(context.Background(), ProposalScope{
		ChainID:         46,
		DaoCode:         "ring-dao",
		GovernorAddress: "0xAbC123",
	}, 0, 1, "power_DESC")
	if err != nil {
		t.Fatalf("QueryContributors() error = %v", err)
	}
	if len(contributors) != 1 {
		t.Fatalf("len(contributors) = %d, want 1", len(contributors))
	}
	if contributors[0].Balance == nil || *contributors[0].Balance != "25" {
		t.Fatalf("balance = %#v, want 25", contributors[0].Balance)
	}
}
