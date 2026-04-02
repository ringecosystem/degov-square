package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQueryGlobalDataMetricsFallsBackToProposalsConnectionWhenGlobalProposalCountUnavailable(t *testing.T) {
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
					if _, exists := where["id_eq"]; exists {
						t.Fatalf("unexpected id_eq in proposal count fallback where: %#v", where)
					}
					_, _ = w.Write([]byte(`{"data":{"proposalsConnection":{"totalCount":10}}}`))
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
