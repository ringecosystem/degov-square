package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	dbmodels "github.com/ringecosystem/degov-square/database/models"
	gqlmodels "github.com/ringecosystem/degov-square/graph/models"
	"github.com/ringecosystem/degov-square/types"
)

type fakeDaoService struct {
	listInput   types.BasicInput[*types.ListDaosInput]
	listDaos    []*gqlmodels.Dao
	listErr     error
	inspectCode string
	inspectDao  *gqlmodels.Dao
	inspectErr  error
}

func (s *fakeDaoService) ListDaos(input types.BasicInput[*types.ListDaosInput]) ([]*gqlmodels.Dao, error) {
	s.listInput = input
	return s.listDaos, s.listErr
}

func (s *fakeDaoService) Inspect(input types.BasicInput[string]) (*gqlmodels.Dao, error) {
	s.inspectCode = input.Input
	return s.inspectDao, s.inspectErr
}

type fakeDaoConfigService struct {
	input   gqlmodels.GetDaoConfigInput
	content string
	err     error
}

func (s *fakeDaoConfigService) RawConfig(input gqlmodels.GetDaoConfigInput) (string, error) {
	s.input = input
	return s.content, s.err
}

func TestListDaosToolReturnsBoundedSummaries(t *testing.T) {
	t.Parallel()

	daoService := &fakeDaoService{
		listDaos: []*gqlmodels.Dao{
			testDao("alpha-dao", "Alpha DAO"),
			testDao("beta-dao", "Beta DAO"),
			testDao("gamma-dao", "Gamma DAO"),
		},
	}
	session, closeSession := newTestMCPSession(t, Config{
		Name:             "degov-square",
		Version:          "test-version",
		DaoService:       daoService,
		DaoConfigService: &fakeDaoConfigService{},
	})
	defer closeSession()

	result, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name: "list_daos",
		Arguments: map[string]any{
			"limit": 2,
			"codes": []string{"alpha-dao", "beta-dao"},
		},
	})
	if err != nil {
		t.Fatalf("CallTool(list_daos) error = %v", err)
	}

	content := structuredContent(t, result)
	if got, want := int(content["count"].(float64)), 2; got != want {
		t.Fatalf("count = %d, want %d", got, want)
	}
	if got, want := int(content["limit"].(float64)), 2; got != want {
		t.Fatalf("limit = %d, want %d", got, want)
	}
	daos := content["daos"].([]any)
	if got, want := len(daos), 2; got != want {
		t.Fatalf("len(daos) = %d, want %d", got, want)
	}
	first := daos[0].(map[string]any)
	if got, want := first["daoCode"], "alpha-dao"; got != want {
		t.Fatalf("daoCode = %v, want %v", got, want)
	}
	if _, ok := first["lastProposal"]; ok {
		t.Fatal("summary exposed lastProposal")
	}
	if daoService.listInput.Input == nil || daoService.listInput.Input.Codes == nil {
		t.Fatal("ListDaos input did not include codes filter")
	}
}

func TestGetDaoToolReturnsDetailWithoutProposalData(t *testing.T) {
	t.Parallel()

	daoService := &fakeDaoService{inspectDao: testDao("alpha-dao", "Alpha DAO")}
	session, closeSession := newTestMCPSession(t, Config{
		Name:             "degov-square",
		Version:          "test-version",
		DaoService:       daoService,
		DaoConfigService: &fakeDaoConfigService{},
	})
	defer closeSession()

	result, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "get_dao",
		Arguments: map[string]any{"daoCode": "alpha-dao"},
	})
	if err != nil {
		t.Fatalf("CallTool(get_dao) error = %v", err)
	}

	content := structuredContent(t, result)
	if got, want := content["daoCode"], "alpha-dao"; got != want {
		t.Fatalf("daoCode = %v, want %v", got, want)
	}
	if got, want := content["metricsCountMembers"], float64(42); got != want {
		t.Fatalf("metricsCountMembers = %v, want %v", got, want)
	}
	if _, ok := content["lastProposal"]; ok {
		t.Fatal("detail exposed lastProposal")
	}
	if got, want := daoService.inspectCode, "alpha-dao"; got != want {
		t.Fatalf("Inspect code = %q, want %q", got, want)
	}
}

func TestGetDaoConfigToolDefaultsToJSON(t *testing.T) {
	t.Parallel()

	configService := &fakeDaoConfigService{content: `{"code":"alpha-dao"}`}
	session, closeSession := newTestMCPSession(t, Config{
		Name:             "degov-square",
		Version:          "test-version",
		DaoService:       &fakeDaoService{},
		DaoConfigService: configService,
	})
	defer closeSession()

	result, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "get_dao_config",
		Arguments: map[string]any{"daoCode": "alpha-dao"},
	})
	if err != nil {
		t.Fatalf("CallTool(get_dao_config) error = %v", err)
	}

	content := structuredContent(t, result)
	if got, want := content["daoCode"], "alpha-dao"; got != want {
		t.Fatalf("daoCode = %v, want %v", got, want)
	}
	if got, want := content["format"], "json"; got != want {
		t.Fatalf("format = %v, want %v", got, want)
	}
	if got, want := content["content"], `{"code":"alpha-dao"}`; got != want {
		t.Fatalf("content = %v, want %v", got, want)
	}
	if configService.input.Format == nil || *configService.input.Format != gqlmodels.ConfigFormatJSON {
		t.Fatalf("RawConfig format = %v, want JSON", configService.input.Format)
	}
}

func TestDaoToolsRejectInvalidDaoCode(t *testing.T) {
	t.Parallel()

	session, closeSession := newTestMCPSession(t, Config{
		Name:             "degov-square",
		Version:          "test-version",
		DaoService:       &fakeDaoService{},
		DaoConfigService: &fakeDaoConfigService{},
	})
	defer closeSession()

	_, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "get_dao",
		Arguments: map[string]any{"daoCode": "../secret"},
	})
	if err == nil {
		t.Fatal("CallTool(get_dao) error = nil, want validation error")
	}
	var rpcErr *jsonrpc.Error
	if !errors.As(err, &rpcErr) {
		t.Fatalf("error type = %T, want jsonrpc.Error", err)
	}
	if got, want := rpcErr.Code, int64(jsonrpc.CodeInvalidParams); got != want {
		t.Fatalf("error code = %d, want %d", got, want)
	}
	if !strings.Contains(rpcErr.Message, "daoCode") {
		t.Fatalf("error message %q does not mention daoCode", rpcErr.Message)
	}
}

func TestDaoToolInputSchemasAreConstrained(t *testing.T) {
	t.Parallel()

	session, closeSession := newTestMCPSession(t, Config{
		Name:             "degov-square",
		Version:          "test-version",
		DaoService:       &fakeDaoService{},
		DaoConfigService: &fakeDaoConfigService{},
	})
	defer closeSession()

	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	listDaosSchema := toolInputSchema(t, result.Tools, "list_daos")
	codes := schemaProperty(t, listDaosSchema, "codes")
	if got := codes["type"]; got != "array" {
		t.Fatalf("list_daos codes type = %v, want array", got)
	}
	codesItems := codes["items"].(map[string]any)
	if got := codesItems["type"]; got != "string" {
		t.Fatalf("list_daos codes item type = %v, want string", got)
	}
	if got := codesItems["pattern"]; got != daoCodePattern.String() {
		t.Fatalf("list_daos codes pattern = %v, want %s", got, daoCodePattern.String())
	}

	state := schemaProperty(t, listDaosSchema, "state")
	if got := state["type"]; got != "string" {
		t.Fatalf("list_daos state type = %v, want string", got)
	}
	assertStringEnum(t, state["enum"], []string{"ACTIVE", "DRAFT", "INACTIVE", "active", "draft", "inactive"})

	getDaoConfigSchema := toolInputSchema(t, result.Tools, "get_dao_config")
	format := schemaProperty(t, getDaoConfigSchema, "format")
	if got := format["type"]; got != "string" {
		t.Fatalf("get_dao_config format type = %v, want string", got)
	}
	assertStringEnum(t, format["enum"], []string{"json", "yaml"})
}

func TestDaoToolsReturnStructuredNotFoundError(t *testing.T) {
	t.Parallel()

	session, closeSession := newTestMCPSession(t, Config{
		Name:             "degov-square",
		Version:          "test-version",
		DaoService:       &fakeDaoService{inspectErr: errors.New("dao not found")},
		DaoConfigService: &fakeDaoConfigService{},
	})
	defer closeSession()

	_, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "get_dao",
		Arguments: map[string]any{"daoCode": "missing-dao"},
	})
	if err == nil {
		t.Fatal("CallTool(get_dao) error = nil, want not-found error")
	}
	var rpcErr *jsonrpc.Error
	if !errors.As(err, &rpcErr) {
		t.Fatalf("error type = %T, want jsonrpc.Error", err)
	}
	if got, want := rpcErr.Code, int64(jsonrpc.CodeInvalidParams); got != want {
		t.Fatalf("error code = %d, want %d", got, want)
	}
	if !strings.Contains(rpcErr.Message, "not found") {
		t.Fatalf("error message %q does not mention not found", rpcErr.Message)
	}
}

func toolInputSchema(t *testing.T, tools []*sdkmcp.Tool, name string) map[string]any {
	t.Helper()

	for _, tool := range tools {
		if tool.Name != name {
			continue
		}
		data, err := json.Marshal(tool.InputSchema)
		if err != nil {
			t.Fatalf("Marshal %s input schema: %v", name, err)
		}
		var schema map[string]any
		if err := json.Unmarshal(data, &schema); err != nil {
			t.Fatalf("Unmarshal %s input schema: %v", name, err)
		}
		return schema
	}
	t.Fatalf("%s not found in tool listing", name)
	return nil
}

func schemaProperty(t *testing.T, schema map[string]any, name string) map[string]any {
	t.Helper()

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema properties type = %T, want map[string]any", schema["properties"])
	}
	property, ok := properties[name].(map[string]any)
	if !ok {
		t.Fatalf("schema property %s type = %T, want map[string]any", name, properties[name])
	}
	return property
}

func assertStringEnum(t *testing.T, value any, want []string) {
	t.Helper()

	values, ok := value.([]any)
	if !ok {
		t.Fatalf("enum type = %T, want []any", value)
	}
	if len(values) != len(want) {
		t.Fatalf("enum len = %d, want %d", len(values), len(want))
	}
	for i, value := range values {
		if value != want[i] {
			t.Fatalf("enum[%d] = %v, want %s", i, value, want[i])
		}
	}
}

func newTestMCPSession(t *testing.T, cfg Config) (*sdkmcp.ClientSession, func()) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()
	server := NewServer(cfg)
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

func structuredContent(t *testing.T, result *sdkmcp.CallToolResult) map[string]any {
	t.Helper()

	content, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("StructuredContent type = %T, want map[string]any", result.StructuredContent)
	}
	return content
}

func testDao(code, name string) *gqlmodels.Dao {
	now := time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC)
	return &gqlmodels.Dao{
		ID:                    "internal-id-" + code,
		ChainID:               1,
		ChainName:             "Ethereum",
		Name:                  name,
		Code:                  code,
		Endpoint:              "https://" + code + ".example",
		State:                 string(dbmodels.DaoStateActive),
		Domains:               []string{code + ".example"},
		Tags:                  []string{"governance"},
		MetricsCountProposals: 7,
		MetricsCountMembers:   42,
		MetricsSumPower:       "1000",
		MetricsCountVote:      21,
		Ctime:                 now,
		LastProposal:          &gqlmodels.Proposal{ID: "proposal-id"},
	}
}
