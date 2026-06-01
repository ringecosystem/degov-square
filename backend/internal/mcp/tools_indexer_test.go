package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/ringecosystem/degov-square/services"
)

func TestListContributorsToolReturnsBoundedIdentities(t *testing.T) {
	requests := 0
	indexer := newMCPIndexerServer(t, `{"data":{"contributors":[{"id":"0x0000000000000000000000000000000000000001","power":"100","balance":"25","delegatesCountAll":2,"delegatesCountEffective":1},{"id":"0x0000000000000000000000000000000000000002","power":"50","balance":"10","delegatesCountAll":0,"delegatesCountEffective":0}]}}`)
	defer indexer.Close()

	server := newTestProposalServer(t)
	seedMCPDaoConfig(t, "ring-dao", indexer.URL)

	result := callProposalTool(t, server, "list_contributors", map[string]any{
		"daoCode": "ring-dao",
		"limit":   999,
	})
	content := requireStructuredContent(t, result)

	if got, want := content["limit"], float64(maxContributorListLimit); got != want {
		t.Fatalf("limit = %v, want %v", got, want)
	}
	contributors := content["contributors"].([]any)
	if got, want := len(contributors), 2; got != want {
		t.Fatalf("len(contributors) = %d, want %d", got, want)
	}
	first := contributors[0].(map[string]any)
	account := first["account"].(map[string]any)
	if got, want := account["address"], "0x0000000000000000000000000000000000000001"; got != want {
		t.Fatalf("account.address = %v, want %v", got, want)
	}
	if got, want := first["delegatesCountAll"], float64(2); got != want {
		t.Fatalf("delegatesCountAll = %v, want %v", got, want)
	}
	if got, want := first["balance"], "25"; got != want {
		t.Fatalf("balance = %v, want %v", got, want)
	}
	if requests == 0 {
		t.Log("indexer request handled by shared test server")
	}
}

func TestGetContributorToolIncludesENSNameWhenAvailable(t *testing.T) {
	name := "alice.eth"
	indexer := newMCPIndexerServer(t, `{"data":{"contributors":[{"id":"0x0000000000000000000000000000000000000001","power":"100","balance":"25","delegatesCountAll":2,"delegatesCountEffective":1}]}}`)
	defer indexer.Close()

	server := newTestProposalServerWithConfig(t, Config{
		Name:    "degov-square",
		Version: "test-version",
		ENSService: &fakeENSService{resolveRecords: map[string]*services.ENSRecord{
			"0x0000000000000000000000000000000000000001": {
				Address: stringPtr("0x0000000000000000000000000000000000000001"),
				Name:    &name,
			},
		}},
	})
	seedMCPDaoConfig(t, "ring-dao", indexer.URL)

	result := callProposalTool(t, server, "get_contributor", map[string]any{
		"daoCode": "ring-dao",
		"address": "0x0000000000000000000000000000000000000001",
	})
	content := requireStructuredContent(t, result)

	contributor := content["contributor"].(map[string]any)
	account := contributor["account"].(map[string]any)
	if got, want := account["ensName"], name; got != want {
		t.Fatalf("account.ensName = %v, want %v", got, want)
	}
}

func TestListProposalVotesToolReturnsVoterIdentities(t *testing.T) {
	indexer := newMCPIndexerServer(t, `{"data":{"voteCasts":[{"id":"vote-1","proposalId":"0xproposal","voter":"0x0000000000000000000000000000000000000001","support":1,"weight":"100","reason":"","transactionHash":"0xtx","blockNumber":"1","blockTimestamp":"2"}]}}`)
	defer indexer.Close()

	server := newTestProposalServer(t)
	seedMCPDaoConfig(t, "ring-dao", indexer.URL)

	result := callProposalTool(t, server, "list_proposal_votes", map[string]any{
		"daoCode":    "ring-dao",
		"proposalId": "0xproposal",
		"limit":      999,
	})
	content := requireStructuredContent(t, result)

	if got, want := content["limit"], float64(maxProposalVotesListLimit); got != want {
		t.Fatalf("limit = %v, want %v", got, want)
	}
	votes := content["votes"].([]any)
	vote := votes[0].(map[string]any)
	voter := vote["voter"].(map[string]any)
	if got, want := voter["address"], "0x0000000000000000000000000000000000000001"; got != want {
		t.Fatalf("voter.address = %v, want %v", got, want)
	}
}

func TestIndexerToolsReturnClearErrors(t *testing.T) {
	server := newTestProposalServer(t)

	tests := []struct {
		name      string
		tool      string
		arguments map[string]any
		wantError string
	}{
		{
			name:      "contributor address required",
			tool:      "get_contributor",
			arguments: map[string]any{"daoCode": "ring-dao", "address": ""},
			wantError: "invalid_address",
		},
		{
			name:      "votes proposal required",
			tool:      "list_proposal_votes",
			arguments: map[string]any{"daoCode": "ring-dao", "proposalId": ""},
			wantError: "invalid_proposal_id",
		},
		{
			name:      "contributors offset",
			tool:      "list_contributors",
			arguments: map[string]any{"daoCode": "ring-dao", "offset": -1},
			wantError: "invalid_offset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := callProposalTool(t, server, tt.tool, tt.arguments)
			requireToolErrorContains(t, result, tt.wantError)
		})
	}
}

func TestIndexerToolAnnotationsAreReadOnly(t *testing.T) {
	server := newTestProposalServer(t)
	session, closeSession := newProposalTestSession(t, server)
	defer closeSession()

	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	for _, toolName := range []string{"get_contributor", "list_contributors", "list_proposal_votes"} {
		found := false
		for _, tool := range result.Tools {
			if tool.Name != toolName {
				continue
			}
			found = true
			if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
				t.Fatalf("%s readOnlyHint = %v, want true", toolName, tool.Annotations)
			}
		}
		if !found {
			t.Fatalf("%s not found in tool listing", toolName)
		}
	}
}

func TestIndexerToolsRejectUnknownDAO(t *testing.T) {
	server := newTestProposalServer(t)

	result := callProposalTool(t, server, "list_contributors", map[string]any{"daoCode": strings.Repeat("missing", 1)})

	requireToolErrorContains(t, result, "dao_not_found")
}
