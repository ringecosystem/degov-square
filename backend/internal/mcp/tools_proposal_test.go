package mcp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ringecosystem/degov-square/database"
	dbmodels "github.com/ringecosystem/degov-square/database/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestProposalServer(t *testing.T) *sdkmcp.Server {
	return newTestProposalServerWithConfig(t, Config{Name: "degov-square", Version: "test-version"})
}

func newTestProposalServerWithConfig(t *testing.T, cfg Config) *sdkmcp.Server {
	t.Helper()

	previousDB := database.DB
	t.Cleanup(func() {
		database.DB = previousDB
	})

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE dgv_dao (
			id TEXT PRIMARY KEY,
			chain_id INTEGER NOT NULL,
			chain_name TEXT NOT NULL,
			chain_logo TEXT,
			name TEXT NOT NULL,
			code TEXT NOT NULL,
			logo TEXT,
			seq INTEGER NOT NULL DEFAULT 0,
			endpoint TEXT NOT NULL,
			state TEXT NOT NULL,
			tags TEXT,
			domains TEXT,
			features TEXT,
			config_link TEXT NOT NULL,
			time_syncd DATETIME,
			metrics_count_proposals INTEGER NOT NULL DEFAULT 0,
			metrics_count_members INTEGER NOT NULL DEFAULT 0,
			metrics_sum_power TEXT NOT NULL DEFAULT '0',
			metrics_count_vote INTEGER NOT NULL DEFAULT 0,
			last_tracked_block_number INTEGER NOT NULL DEFAULT 0,
			last_tracked_proposal_id TEXT NOT NULL DEFAULT '',
			ctime DATETIME NOT NULL,
			utime DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create dao table: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE dgv_proposal_tracking (
			id TEXT PRIMARY KEY,
			dao_code TEXT NOT NULL,
			chain_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			proposal_link TEXT NOT NULL,
			proposal_id TEXT NOT NULL,
			state TEXT NOT NULL,
			proposal_created_at DATETIME,
			proposal_at_block INTEGER NOT NULL,
			times_track INTEGER NOT NULL DEFAULT 0,
			time_next_track DATETIME,
			message TEXT,
			offset_tracking_vote INTEGER DEFAULT 0,
			fulfilled INTEGER DEFAULT 0,
			fulfilled_explain TEXT,
			fulfilled_at DATETIME,
			times_fulfill INTEGER DEFAULT 0,
			fulfill_errored INTEGER DEFAULT 0,
			ctime DATETIME NOT NULL,
			utime DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create proposal tracking table: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE dgv_proposal_summary (
			id TEXT PRIMARY KEY,
			dao_code TEXT,
			chain_id INTEGER NOT NULL,
			proposal_id TEXT NOT NULL,
			indexer TEXT,
			description TEXT NOT NULL,
			summary TEXT NOT NULL,
			ctime DATETIME NOT NULL,
			utime DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create proposal summary table: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE dgv_dao_config (
			id TEXT PRIMARY KEY,
			dao_code TEXT NOT NULL,
			config TEXT NOT NULL,
			ctime DATETIME NOT NULL,
			utime DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create dao config table: %v", err)
	}
	database.DB = db

	seedMCPDao(t, db, "ring-dao")
	if cfg.Name == "" {
		cfg.Name = "degov-square"
	}
	if cfg.Version == "" {
		cfg.Version = "test-version"
	}
	return NewServer(cfg)
}

func seedMCPDao(t *testing.T, db *gorm.DB, code string) {
	t.Helper()

	if err := db.Create(&dbmodels.Dao{
		ID:         "dao-" + code,
		ChainID:    46,
		ChainName:  "Darwinia",
		Name:       "Ring DAO",
		Code:       code,
		Endpoint:   "https://gov.ringdao.com",
		State:      dbmodels.DaoStateActive,
		ConfigLink: "https://example.com/config.yml",
		CTime:      time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed dao: %v", err)
	}
}

func seedMCPProposal(t *testing.T, proposal dbmodels.ProposalTracking) {
	t.Helper()

	if err := database.DB.Create(&proposal).Error; err != nil {
		t.Fatalf("seed proposal: %v", err)
	}
}

func seedMCPDaoConfig(t *testing.T, daoCode string, indexerEndpoint string) {
	t.Helper()

	config := fmt.Sprintf("name: Ring DAO\ncode: %s\nindexer:\n  endpoint: %s\ncontracts:\n  governor: \"0x52cdd25f7c83c335236ce209fa1ec8e197e96533\"\nchain:\n  id: 46\n", daoCode, indexerEndpoint)
	if err := database.DB.Exec(`
		INSERT INTO dgv_dao_config (id, dao_code, config, ctime)
		VALUES (?, ?, ?, ?)
	`, "config-"+daoCode, daoCode, config, time.Now()).Error; err != nil {
		t.Fatalf("seed dao config: %v", err)
	}
}

func newMCPIndexerServer(t *testing.T, response string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
}

func callProposalTool(t *testing.T, server *sdkmcp.Server, name string, arguments map[string]any) *sdkmcp.CallToolResult {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()
	go func() {
		_ = server.Run(ctx, serverTransport)
	}()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test-client"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      name,
		Arguments: arguments,
	})
	if err != nil {
		t.Fatalf("CallTool() protocol error = %v", err)
	}
	return result
}

func requireStructuredContent(t *testing.T, result *sdkmcp.CallToolResult) map[string]any {
	t.Helper()

	if result.IsError {
		t.Fatalf("CallTool() returned error result: %v", result.Content)
	}
	content, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("StructuredContent type = %T, want map[string]any", result.StructuredContent)
	}
	return content
}

func requireToolErrorContains(t *testing.T, result *sdkmcp.CallToolResult, want string) {
	t.Helper()

	if !result.IsError {
		t.Fatalf("IsError = false, want true")
	}
	if len(result.Content) == 0 {
		t.Fatalf("error content is empty")
	}
	text, ok := result.Content[0].(*sdkmcp.TextContent)
	if !ok {
		t.Fatalf("error content type = %T, want *TextContent", result.Content[0])
	}
	if !strings.Contains(text.Text, want) {
		t.Fatalf("error text = %q, want substring %q", text.Text, want)
	}
}

func TestListProposalsToolReturnsBoundedRows(t *testing.T) {
	server := newTestProposalServer(t)
	createdAt := time.Now().Add(-time.Hour)
	for i := 0; i < 60; i++ {
		seedMCPProposal(t, dbmodels.ProposalTracking{
			ID:                fmt.Sprintf("proposal-%d", i),
			DaoCode:           "ring-dao",
			ChainId:           46,
			Title:             fmt.Sprintf("Proposal %d", i),
			ProposalLink:      fmt.Sprintf("https://gov.ringdao.com/proposal/%d", i),
			ProposalID:        fmt.Sprintf("0x%x", i),
			State:             dbmodels.ProposalStateActive,
			ProposalCreatedAt: &createdAt,
			CTime:             time.Now(),
		})
	}

	result := callProposalTool(t, server, "list_proposals", map[string]any{
		"daoCode": "ring-dao",
		"limit":   99,
	})
	content := requireStructuredContent(t, result)

	if got, want := content["limit"], float64(50); got != want {
		t.Fatalf("limit = %v, want %v", got, want)
	}
	proposals, ok := content["proposals"].([]any)
	if !ok {
		t.Fatalf("proposals type = %T, want []any", content["proposals"])
	}
	if len(proposals) != 50 {
		t.Fatalf("len(proposals) = %d, want 50", len(proposals))
	}
}

func TestListProposalsToolFiltersByState(t *testing.T) {
	server := newTestProposalServer(t)

	seedMCPProposal(t, dbmodels.ProposalTracking{
		ID:           "proposal-active",
		DaoCode:      "ring-dao",
		ChainId:      46,
		Title:        "Active proposal",
		ProposalLink: "https://gov.ringdao.com/proposal/active",
		ProposalID:   "0xactive",
		State:        dbmodels.ProposalStateActive,
		CTime:        time.Now(),
	})
	seedMCPProposal(t, dbmodels.ProposalTracking{
		ID:           "proposal-executed",
		DaoCode:      "ring-dao",
		ChainId:      46,
		Title:        "Executed proposal",
		ProposalLink: "https://gov.ringdao.com/proposal/executed",
		ProposalID:   "0xexecuted",
		State:        dbmodels.ProposalStateExecuted,
		CTime:        time.Now(),
	})

	result := callProposalTool(t, server, "list_proposals", map[string]any{
		"daoCode": "ring-dao",
		"state":   "executed",
	})
	content := requireStructuredContent(t, result)
	proposals := content["proposals"].([]any)

	if len(proposals) != 1 {
		t.Fatalf("len(proposals) = %d, want 1", len(proposals))
	}
	proposal := proposals[0].(map[string]any)
	if got, want := proposal["proposalId"], "0xexecuted"; got != want {
		t.Fatalf("proposalId = %v, want %v", got, want)
	}
}

func TestGetProposalToolReturnsDetail(t *testing.T) {
	server := newTestProposalServer(t)
	createdAt := time.Now().Add(-time.Hour)

	seedMCPProposal(t, dbmodels.ProposalTracking{
		ID:                 "proposal-detail",
		DaoCode:            "ring-dao",
		ChainId:            46,
		Title:              "Detailed proposal",
		ProposalLink:       "https://gov.ringdao.com/proposal/detail",
		ProposalID:         "0xdetail",
		State:              dbmodels.ProposalStateSucceeded,
		ProposalCreatedAt:  &createdAt,
		ProposalAtBlock:    12345,
		OffsetTrackingVote: 12,
		CTime:              time.Now(),
	})

	result := callProposalTool(t, server, "get_proposal", map[string]any{
		"daoCode":    "ring-dao",
		"proposalId": "0xdetail",
	})
	content := requireStructuredContent(t, result)

	proposal := content["proposal"].(map[string]any)
	if got, want := proposal["proposalId"], "0xdetail"; got != want {
		t.Fatalf("proposalId = %v, want %v", got, want)
	}
	if got, want := proposal["state"], "SUCCEEDED"; got != want {
		t.Fatalf("state = %v, want %v", got, want)
	}
	if got, want := proposal["offsetTrackingVote"], float64(12); got != want {
		t.Fatalf("offsetTrackingVote = %v, want %v", got, want)
	}
}

func TestProposalToolsListIncludesGetProposalState(t *testing.T) {
	server := newTestProposalServer(t)
	session, closeSession := newProposalTestSession(t, server)
	defer closeSession()

	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	for _, tool := range result.Tools {
		if tool.Name == "get_proposal_state" {
			return
		}
	}
	t.Fatal("get_proposal_state not found in tool listing")
}

func TestProposalToolInputSchemasAreConstrained(t *testing.T) {
	server := newTestProposalServer(t)
	session, closeSession := newProposalTestSession(t, server)
	defer closeSession()

	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	listProposalsSchema := toolInputSchema(t, result.Tools, "list_proposals")
	state := schemaProperty(t, listProposalsSchema, "state")
	if got := state["type"]; got != "string" {
		t.Fatalf("list_proposals state type = %v, want string", got)
	}
	assertStringEnum(t, state["enum"], []string{
		"UNKNOWN",
		"PENDING",
		"ACTIVE",
		"CANCELED",
		"DEFEATED",
		"SUCCEEDED",
		"QUEUED",
		"EXECUTED",
		"EXPIRED",
		"unknown",
		"pending",
		"active",
		"canceled",
		"defeated",
		"succeeded",
		"queued",
		"executed",
		"expired",
	})
}

func TestProposalToolsListOmitsStandaloneENSTools(t *testing.T) {
	server := newTestProposalServer(t)
	session, closeSession := newProposalTestSession(t, server)
	defer closeSession()

	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	for _, tool := range result.Tools {
		if tool.Name == "resolve_ens" || tool.Name == "resolve_ens_records" {
			t.Fatalf("unexpected standalone ENS tool listed: %s", tool.Name)
		}
	}
}

func TestGetProposalToolIncludesBestEffortProposerIdentity(t *testing.T) {
	indexer := newMCPIndexerServer(t, `{"data":{"proposals":[{"id":"indexer-proposal","proposalId":"0xdetail","proposer":"0x0000000000000000000000000000000000000001"}]}}`)
	defer indexer.Close()

	server := newTestProposalServer(t)
	seedMCPDaoConfig(t, "ring-dao", indexer.URL)
	createdAt := time.Now().Add(-time.Hour)
	seedMCPProposal(t, dbmodels.ProposalTracking{
		ID:                "proposal-detail-identity",
		DaoCode:           "ring-dao",
		ChainId:           46,
		Title:             "Detailed proposal",
		ProposalLink:      "https://gov.ringdao.com/proposal/detail",
		ProposalID:        "0xdetail",
		State:             dbmodels.ProposalStateSucceeded,
		ProposalCreatedAt: &createdAt,
		CTime:             time.Now(),
	})

	result := callProposalTool(t, server, "get_proposal", map[string]any{
		"daoCode":    "ring-dao",
		"proposalId": "0xdetail",
	})
	content := requireStructuredContent(t, result)

	proposal := content["proposal"].(map[string]any)
	proposer := proposal["proposer"].(map[string]any)
	if got, want := proposer["address"], "0x0000000000000000000000000000000000000001"; got != want {
		t.Fatalf("proposer.address = %v, want %v", got, want)
	}
	if _, ok := proposer["ensName"]; ok {
		t.Fatalf("proposer.ensName present without ENS result")
	}
}

func TestGetProposalStateToolReturnsTrackedState(t *testing.T) {
	server := newTestProposalServer(t)
	createdAt := time.Now().Add(-time.Hour).UTC()
	updatedAt := time.Now().UTC()

	seedMCPProposal(t, dbmodels.ProposalTracking{
		ID:                "proposal-state",
		DaoCode:           "ring-dao",
		ChainId:           46,
		Title:             "State proposal",
		ProposalLink:      "https://gov.ringdao.com/proposal/state",
		ProposalID:        "0xstate",
		State:             dbmodels.ProposalStateQueued,
		ProposalCreatedAt: &createdAt,
		ProposalAtBlock:   23456,
		CTime:             createdAt,
		UTime:             &updatedAt,
	})

	result := callProposalTool(t, server, "get_proposal_state", map[string]any{
		"daoCode":    " ring-dao ",
		"proposalId": " 0xstate ",
	})
	content := requireStructuredContent(t, result)

	if got, want := content["daoCode"], "ring-dao"; got != want {
		t.Fatalf("daoCode = %v, want %v", got, want)
	}
	if got, want := content["proposalId"], "0xstate"; got != want {
		t.Fatalf("proposalId = %v, want %v", got, want)
	}
	if got, want := content["state"], "QUEUED"; got != want {
		t.Fatalf("state = %v, want %v", got, want)
	}
	if got, want := content["source"], "tracked"; got != want {
		t.Fatalf("source = %v, want %v", got, want)
	}
	if _, ok := content["updatedAt"]; !ok {
		t.Fatal("updatedAt is missing")
	}
}

func TestGetProposalStateToolReturnsClearErrors(t *testing.T) {
	tests := []struct {
		name      string
		arguments map[string]any
		wantError string
	}{
		{
			name:      "invalid dao code",
			arguments: map[string]any{"daoCode": "", "proposalId": "0xstate"},
			wantError: "invalid_dao_code",
		},
		{
			name:      "invalid proposal id",
			arguments: map[string]any{"daoCode": "ring-dao", "proposalId": ""},
			wantError: "invalid_proposal_id",
		},
		{
			name:      "unknown dao",
			arguments: map[string]any{"daoCode": "missing-dao", "proposalId": "0xstate"},
			wantError: "dao_not_found",
		},
		{
			name:      "unknown proposal",
			arguments: map[string]any{"daoCode": "ring-dao", "proposalId": "0xmissing"},
			wantError: "proposal_not_found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestProposalServer(t)

			result := callProposalTool(t, server, "get_proposal_state", tt.arguments)

			requireToolErrorContains(t, result, tt.wantError)
		})
	}
}

func TestGetProposalStateToolRejectsUnavailableState(t *testing.T) {
	server := newTestProposalServer(t)
	seedMCPProposal(t, dbmodels.ProposalTracking{
		ID:           "proposal-state-unavailable",
		DaoCode:      "ring-dao",
		ChainId:      46,
		Title:        "State unavailable proposal",
		ProposalLink: "https://gov.ringdao.com/proposal/state-unavailable",
		ProposalID:   "0xnostate",
		State:        "",
		CTime:        time.Now(),
	})

	result := callProposalTool(t, server, "get_proposal_state", map[string]any{
		"daoCode":    "ring-dao",
		"proposalId": "0xnostate",
	})

	requireToolErrorContains(t, result, "proposal_state_unavailable")
}

func TestGetProposalStateToolDoesNotMutateTrackingState(t *testing.T) {
	server := newTestProposalServer(t)
	nextTrackAt := time.Now().Add(time.Hour).UTC()
	updatedAt := time.Now().Add(-time.Minute).UTC()
	seedMCPProposal(t, dbmodels.ProposalTracking{
		ID:                 "proposal-read-only",
		DaoCode:            "ring-dao",
		ChainId:            46,
		Title:              "Read-only proposal",
		ProposalLink:       "https://gov.ringdao.com/proposal/read-only",
		ProposalID:         "0xreadonly",
		State:              dbmodels.ProposalStateActive,
		TimesTrack:         5,
		TimeNextTrack:      &nextTrackAt,
		Message:            "keep retry metadata",
		OffsetTrackingVote: 9,
		CTime:              time.Now().Add(-time.Hour).UTC(),
		UTime:              &updatedAt,
	})

	result := callProposalTool(t, server, "get_proposal_state", map[string]any{
		"daoCode":    "ring-dao",
		"proposalId": "0xreadonly",
	})
	requireStructuredContent(t, result)

	var proposal dbmodels.ProposalTracking
	if err := database.DB.Where("dao_code = ? AND proposal_id = ?", "ring-dao", "0xreadonly").First(&proposal).Error; err != nil {
		t.Fatalf("load proposal: %v", err)
	}
	if got, want := proposal.TimesTrack, 5; got != want {
		t.Fatalf("TimesTrack = %d, want %d", got, want)
	}
	if proposal.TimeNextTrack == nil || !proposal.TimeNextTrack.Equal(nextTrackAt) {
		t.Fatalf("TimeNextTrack = %v, want %v", proposal.TimeNextTrack, nextTrackAt)
	}
	if got, want := proposal.Message, "keep retry metadata"; got != want {
		t.Fatalf("Message = %q, want %q", got, want)
	}
	if got, want := proposal.OffsetTrackingVote, 9; got != want {
		t.Fatalf("OffsetTrackingVote = %d, want %d", got, want)
	}
	if proposal.UTime == nil || !proposal.UTime.Equal(updatedAt) {
		t.Fatalf("UTime = %v, want %v", proposal.UTime, updatedAt)
	}
}

func TestProposalToolsReturnClearErrors(t *testing.T) {
	tests := []struct {
		name      string
		tool      string
		arguments map[string]any
		wantError string
	}{
		{
			name:      "invalid state",
			tool:      "list_proposals",
			arguments: map[string]any{"daoCode": "ring-dao", "state": "bad"},
			wantError: "does not equal any of",
		},
		{
			name:      "invalid dao code",
			tool:      "list_proposals",
			arguments: map[string]any{"daoCode": ""},
			wantError: "invalid_dao_code",
		},
		{
			name:      "unknown dao",
			tool:      "list_proposals",
			arguments: map[string]any{"daoCode": "missing-dao"},
			wantError: "dao_not_found",
		},
		{
			name:      "invalid proposal id",
			tool:      "get_proposal",
			arguments: map[string]any{"daoCode": "ring-dao", "proposalId": ""},
			wantError: "invalid_proposal_id",
		},
		{
			name:      "unknown proposal",
			tool:      "get_proposal",
			arguments: map[string]any{"daoCode": "ring-dao", "proposalId": "0xmissing"},
			wantError: "proposal_not_found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestProposalServer(t)

			result := callProposalTool(t, server, tt.tool, tt.arguments)

			requireToolErrorContains(t, result, tt.wantError)
		})
	}
}

func newProposalTestSession(t *testing.T, server *sdkmcp.Server) (*sdkmcp.ClientSession, func()) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()
	go func() {
		_ = server.Run(ctx, serverTransport)
	}()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test-client"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		cancel()
		t.Fatalf("Connect() error = %v", err)
	}

	return session, func() {
		session.Close()
		cancel()
	}
}
