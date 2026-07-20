package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestQueryProposalsByBlockNumberRequestsOneServerCursorBatch(t *testing.T) {
	type graphqlRequest struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}

	longID := strings.Repeat("x", 369)
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var req graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if strings.Contains(req.Query, "offset:") {
			t.Fatalf("query = %s, want no all-history offset scan", req.Query)
		}
		if !strings.Contains(req.Query, "orderBy: [blockNumber_ASC_NULLS_FIRST, id_ASC]") {
			t.Fatalf("query = %s, want provider cursor order", req.Query)
		}
		where, ok := req.Variables["where"].(map[string]any)
		if !ok {
			t.Fatalf("where = %#v, want map", req.Variables["where"])
		}
		if got, want := req.Variables["limit"], float64(30); got != want {
			t.Fatalf("limit = %#v, want %#v", got, want)
		}
		cursor, ok := where["OR"].([]any)
		if !ok || len(cursor) != 2 {
			t.Fatalf("OR = %#v, want two cursor branches", where["OR"])
		}
		nextBlock := cursor[0].(map[string]any)
		if got, want := nextBlock["blockNumber_gt"], "100"; got != want {
			t.Fatalf("blockNumber_gt = %#v, want string %#v", got, want)
		}
		sameBlock := cursor[1].(map[string]any)
		if got, want := sameBlock["blockNumber_eq"], "100"; got != want {
			t.Fatalf("blockNumber_eq = %#v, want string %#v", got, want)
		}
		if got, want := sameBlock["id_gt"], "m"; got != want {
			t.Fatalf("id_gt = %#v, want %#v", got, want)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"proposals": []Proposal{
			{ID: "z", BlockNumber: "100"},
			{ID: longID, BlockNumber: "101"},
			{ID: "later-block", BlockNumber: "102"},
		}}})
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
	if got, want := requestCount, 1; got != want {
		t.Fatalf("request count = %d, want %d", got, want)
	}
	if got, want := len(proposals), 3; got != want {
		t.Fatalf("len(proposals) = %d, want %d: %#v", got, want, proposals)
	}
	if got, want := []string{proposals[0].ID, proposals[1].ID, proposals[2].ID}, []string{"z", longID, "later-block"}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("proposal ids = %#v, want %#v", got, want)
	}
}

func TestQueryProposalsByBlockNumberContinuesBeyondFullBatch(t *testing.T) {
	type graphqlRequest struct {
		Variables map[string]any `json:"variables"`
	}

	longID := strings.Repeat("z", 369)
	firstBatch := []Proposal{{ID: "same-block-z", BlockNumber: "100"}}
	for i := 0; i < 28; i++ {
		firstBatch = append(firstBatch, Proposal{ID: fmt.Sprintf("id-%02d", i), BlockNumber: "101"})
	}
	firstBatch = append(firstBatch, Proposal{ID: longID, BlockNumber: "101"})
	secondBatch := []Proposal{
		{ID: longID + "z", BlockNumber: "101"},
		{ID: "later-block", BlockNumber: "102"},
	}

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		where := req.Variables["where"].(map[string]any)
		cursor := where["OR"].([]any)
		nextBlock := cursor[0].(map[string]any)
		sameBlock := cursor[1].(map[string]any)

		response := firstBatch
		if requestCount == 0 {
			if got, want := nextBlock["blockNumber_gt"], "100"; got != want {
				t.Fatalf("first blockNumber_gt = %#v, want %#v", got, want)
			}
			if got, want := sameBlock["blockNumber_eq"], "100"; got != want {
				t.Fatalf("first blockNumber_eq = %#v, want %#v", got, want)
			}
			if got, want := sameBlock["id_gt"], "same-block-m"; got != want {
				t.Fatalf("first id_gt = %#v, want %#v", got, want)
			}
		} else {
			response = secondBatch
			if got, want := nextBlock["blockNumber_gt"], "101"; got != want {
				t.Fatalf("second blockNumber_gt = %#v, want string %#v", got, want)
			}
			if got, want := sameBlock["blockNumber_eq"], "101"; got != want {
				t.Fatalf("second blockNumber_eq = %#v, want string %#v", got, want)
			}
			if got, want := sameBlock["id_gt"], longID; got != want {
				t.Fatalf("second id_gt length = %d, want preserved length %d", len(fmt.Sprint(got)), len(want))
			}
		}
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"proposals": response}})
	}))
	defer server.Close()

	indexer := NewDegovIndexer(server.URL)
	first, err := indexer.QueryProposalsByBlockNumber(ProposalScope{DaoCode: "ring-dao"}, 100, "same-block-m")
	if err != nil {
		t.Fatalf("first QueryProposalsByBlockNumber() error = %v", err)
	}
	if got, want := len(first), 30; got != want {
		t.Fatalf("first batch length = %d, want %d", got, want)
	}
	last := first[len(first)-1]
	lastBlockNumber, err := strconv.ParseInt(last.BlockNumber, 10, 64)
	if err != nil {
		t.Fatalf("parse last block number: %v", err)
	}
	second, err := indexer.QueryProposalsByBlockNumber(ProposalScope{DaoCode: "ring-dao"}, lastBlockNumber, last.ID)
	if err != nil {
		t.Fatalf("second QueryProposalsByBlockNumber() error = %v", err)
	}
	if got, want := requestCount, 2; got != want {
		t.Fatalf("request count = %d, want %d", got, want)
	}

	combined := append(append([]Proposal(nil), first...), second...)
	want := append(append([]Proposal(nil), firstBatch...), secondBatch...)
	if got := len(combined); got != len(want) {
		t.Fatalf("combined length = %d, want %d", got, len(want))
	}
	seen := make(map[string]struct{}, len(combined))
	for i := range combined {
		if combined[i].ID != want[i].ID || combined[i].BlockNumber != want[i].BlockNumber {
			t.Fatalf("combined[%d] = (%q, %q), want (%q, %q)", i, combined[i].BlockNumber, combined[i].ID, want[i].BlockNumber, want[i].ID)
		}
		if _, exists := seen[combined[i].ID]; exists {
			t.Fatalf("duplicate proposal id %q", combined[i].ID)
		}
		seen[combined[i].ID] = struct{}{}
	}
}

func TestQueryProposalsByBlockNumberRejectsInvalidServerOrder(t *testing.T) {
	tests := []struct {
		name      string
		proposals []Proposal
	}{
		{name: "malformed block", proposals: []Proposal{{ID: "z", BlockNumber: "bad"}}},
		{name: "cursor regression", proposals: []Proposal{{ID: "z", BlockNumber: "99"}}},
		{name: "same block id regression", proposals: []Proposal{{ID: "z", BlockNumber: "101"}, {ID: "a", BlockNumber: "101"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"proposals": tt.proposals}})
			}))
			defer server.Close()

			_, err := NewDegovIndexer(server.URL).QueryProposalsByBlockNumber(ProposalScope{DaoCode: "ring-dao"}, 100, "m")
			if err == nil {
				t.Fatal("QueryProposalsByBlockNumber() error = nil, want invalid order error")
			}
		})
	}
}

func TestQueryExpiringProposalsUsesServerWindowAndPaginates(t *testing.T) {
	type graphqlRequest struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}

	now := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)
	page := make([]Proposal, 50)
	for i := range page {
		page[i] = Proposal{ID: fmt.Sprintf("proposal-%02d", i), VoteEndTimestamp: fmt.Sprint(now.Add(time.Hour).UnixMilli())}
	}

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(req.Query, "orderBy: [blockTimestamp_ASC_NULLS_FIRST, id_ASC]") {
			t.Fatalf("query = %s, want provider expiry order", req.Query)
		}
		where, ok := req.Variables["where"].(map[string]any)
		if !ok {
			t.Fatalf("where = %#v, want map", req.Variables["where"])
		}
		if got, want := where["voteEndTimestamp_gte"], fmt.Sprint(now.UnixMilli()); got != want {
			t.Fatalf("voteEndTimestamp_gte = %#v, want string %#v", got, want)
		}
		if got, want := where["voteEndTimestamp_lt"], fmt.Sprint(now.Add(48*time.Hour).UnixMilli()); got != want {
			t.Fatalf("voteEndTimestamp_lt = %#v, want string %#v", got, want)
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
				page[49],
				{ID: "proposal-50", VoteEndTimestamp: fmt.Sprint(now.Add(2 * time.Hour).UnixMilli())},
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
	if got, want := len(proposals), 51; got != want {
		t.Fatalf("len(proposals) = %d, want %d", got, want)
	}
	if got, want := proposals[50].ID, "proposal-50"; got != want {
		t.Fatalf("last proposal id = %q, want %q", got, want)
	}
}

func TestQueryExpiringProposalsRejectsFullPageWithoutProgress(t *testing.T) {
	firstPage := make([]Proposal, 50)
	for i := range firstPage {
		firstPage[i] = Proposal{ID: fmt.Sprintf("proposal-%02d", i)}
	}

	tests := []struct {
		name       string
		secondPage []Proposal
	}{
		{name: "repeated page", secondPage: append([]Proposal(nil), firstPage...)},
		{name: "same ids reordered", secondPage: append(append([]Proposal(nil), firstPage[1:]...), firstPage[0])},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				page := firstPage
				if requestCount > 0 {
					page = tt.secondPage
				}
				requestCount++
				_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"proposals": page}})
			}))
			defer server.Close()

			_, err := NewDegovIndexer(server.URL).QueryExpiringProposals(ProposalScope{DaoCode: "ring-dao"})
			if err == nil {
				t.Fatal("QueryExpiringProposals() error = nil, want pagination progress error")
			}
			if got, want := requestCount, 2; got != want {
				t.Fatalf("request count = %d, want %d", got, want)
			}
		})
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
		if !strings.Contains(req.Query, "proposals(orderBy: [id_ASC], limit: 2, where: $where)") {
			t.Fatalf("query = %s, want deterministic bounded parent query", req.Query)
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

func TestQueryVotesValidatesProposalParentCardinality(t *testing.T) {
	tests := []struct {
		name          string
		response      string
		wantError     bool
		wantVoteCount int
	}{
		{name: "zero parents", response: `{"data":{"proposals":[]}}`, wantVoteCount: 0},
		{name: "multiple parents", response: `{"data":{"proposals":[{"voters":[]},{"voters":[]}]}}`, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()

			votes, err := NewDegovIndexer(server.URL).QueryVotes(context.Background(), ProposalScope{DaoCode: "ring-dao"}, 0, 30, "proposal-1")
			if tt.wantError {
				if err == nil {
					t.Fatal("QueryVotes() error = nil, want multiple parent error")
				}
				return
			}
			if err != nil {
				t.Fatalf("QueryVotes() error = %v", err)
			}
			if got := len(votes); got != tt.wantVoteCount {
				t.Fatalf("len(votes) = %d, want %d", got, tt.wantVoteCount)
			}
		})
	}
}

func TestQueryVoteUsesDirectNestedIDFilter(t *testing.T) {
	type graphqlRequest struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var req graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if strings.Contains(req.Query, "voteCasts") || strings.Contains(req.Query, "VoteCastWhereInput") {
			t.Fatalf("query uses removed top-level vote shape: %s", req.Query)
		}
		if !strings.Contains(req.Query, "query QueryVote($where: ProposalWhereInput!, $voterWhere: VoteCastGroupWhereInput!)") {
			t.Fatalf("query = %s, want approved voter where input", req.Query)
		}
		if !strings.Contains(req.Query, "proposals(orderBy: [id_ASC], limit: 2, where: $where)") {
			t.Fatalf("query = %s, want deterministic parent lookup", req.Query)
		}
		if !strings.Contains(req.Query, "voters(where: $voterWhere, orderBy: [id_ASC], limit: 2)") {
			t.Fatalf("query = %s, want direct bounded vote lookup", req.Query)
		}
		if _, exists := req.Variables["offset"]; exists {
			t.Fatalf("unexpected offset scan variable: %#v", req.Variables)
		}
		where := req.Variables["where"].(map[string]any)
		if got, want := where["proposalId_eq"], "proposal-1"; got != want {
			t.Fatalf("proposalId_eq = %#v, want %#v", got, want)
		}
		voterWhere := req.Variables["voterWhere"].(map[string]any)
		if got, want := voterWhere["id_eq"], "target-vote"; got != want {
			t.Fatalf("id_eq = %#v, want %#v", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"proposals":[{"voters":[{"id":"target-vote","voter":"0x1"}]}]}}`))
	}))
	defer server.Close()

	vote, err := NewDegovIndexer(server.URL).QueryVote(ProposalScope{DaoCode: "ring-dao"}, "proposal-1", "target-vote")
	if err != nil {
		t.Fatalf("QueryVote() error = %v", err)
	}
	if got, want := requestCount, 1; got != want {
		t.Fatalf("request count = %d, want %d", got, want)
	}
	if got, want := vote.ID, "target-vote"; got != want {
		t.Fatalf("vote id = %q, want %q", got, want)
	}
	if got, want := vote.ProposalID, "proposal-1"; got != want {
		t.Fatalf("proposal id = %q, want %q", got, want)
	}
}

func TestQueryVoteValidatesDirectLookupCardinality(t *testing.T) {
	tests := []struct {
		name      string
		response  string
		wantError string
	}{
		{name: "zero parents", response: `{"data":{"proposals":[]}}`, wantError: "no vote found"},
		{name: "zero votes", response: `{"data":{"proposals":[{"voters":[]}]}}`, wantError: "no vote found"},
		{name: "multiple parents", response: `{"data":{"proposals":[{"voters":[]},{"voters":[]}]}}`, wantError: "multiple proposals"},
		{name: "duplicate exact votes", response: `{"data":{"proposals":[{"voters":[{"id":"target-vote"},{"id":"target-vote"}]}]}}`, wantError: "multiple votes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()

			_, err := NewDegovIndexer(server.URL).QueryVote(ProposalScope{DaoCode: "ring-dao"}, "proposal-1", "target-vote")
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("QueryVote() error = %v, want %q", err, tt.wantError)
			}
			if got, want := requestCount, 1; got != want {
				t.Fatalf("request count = %d, want %d", got, want)
			}
		})
	}
}

func TestQueryVoteByVoterUsesNestedScopedFilter(t *testing.T) {
	type graphqlRequest struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if strings.Contains(req.Query, "voteCasts") || strings.Contains(req.Query, "VoteCastWhereInput") {
			t.Fatalf("query uses removed vote shape: %s", req.Query)
		}
		if !strings.Contains(req.Query, "proposals(orderBy: [id_ASC], limit: 2, where: $where)") {
			t.Fatalf("query = %s, want deterministic parent query", req.Query)
		}
		if !strings.Contains(req.Query, "voters(where: $voterWhere, orderBy: [blockTimestamp_ASC_NULLS_LAST, id_ASC], limit: 1)") {
			t.Fatalf("query = %s, want filtered nested voter", req.Query)
		}
		where := req.Variables["where"].(map[string]any)
		if got, want := where["proposalId_eq"], "proposal-1"; got != want {
			t.Fatalf("proposalId_eq = %#v, want %#v", got, want)
		}
		voterWhere := req.Variables["voterWhere"].(map[string]any)
		if got, want := voterWhere["voter_eq"], "0xVoter"; got != want {
			t.Fatalf("voter_eq = %#v, want %#v", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"proposals":[{"voters":[{"id":"vote-1","voter":"0xVoter"}]}]}}`))
	}))
	defer server.Close()

	vote, err := NewDegovIndexer(server.URL).QueryVoteByVoter(ProposalScope{DaoCode: "ring-dao"}, "proposal-1", "0xVoter")
	if err != nil {
		t.Fatalf("QueryVoteByVoter() error = %v", err)
	}
	if got, want := vote.ProposalID, "proposal-1"; got != want {
		t.Fatalf("proposal id = %q, want %q", got, want)
	}
}

func TestQueryVoteByVoterValidatesProposalParentCardinality(t *testing.T) {
	tests := []struct {
		name     string
		response string
	}{
		{name: "zero parents", response: `{"data":{"proposals":[]}}`},
		{name: "multiple parents", response: `{"data":{"proposals":[{"voters":[]},{"voters":[]}]}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()

			_, err := NewDegovIndexer(server.URL).QueryVoteByVoter(ProposalScope{DaoCode: "ring-dao"}, "proposal-1", "0xVoter")
			if err == nil {
				t.Fatal("QueryVoteByVoter() error = nil, want parent cardinality error")
			}
		})
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
