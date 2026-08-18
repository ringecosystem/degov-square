package services

import (
	"encoding/json"
	"strings"
	"testing"

	dbmodels "github.com/ringecosystem/degov-square/database/models"
	gqlmodels "github.com/ringecosystem/degov-square/graph/models"
	"gopkg.in/yaml.v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestProposalCursorPreservesLongIndexerID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE dgv_dao (
			code TEXT PRIMARY KEY,
			last_tracked_block_number INTEGER NOT NULL DEFAULT 0,
			last_tracked_proposal_id TEXT NOT NULL DEFAULT ''
		)
	`).Error; err != nil {
		t.Fatalf("create dao table: %v", err)
	}
	if err := db.Exec("INSERT INTO dgv_dao (code) VALUES (?)", "ring-dao").Error; err != nil {
		t.Fatalf("seed dao: %v", err)
	}

	service := &DaoService{db: db}
	proposalID := strings.Repeat("proposal-id-", 34)
	if len(proposalID) <= 255 {
		t.Fatalf("test proposal id length = %d, want > 255", len(proposalID))
	}
	if err := service.UpdateDaoLastTrackedProposalCursor("ring-dao", 123, proposalID); err != nil {
		t.Fatalf("UpdateDaoLastTrackedProposalCursor() error = %v", err)
	}
	blockNumber, storedProposalID, err := service.GetLastTrackedProposalCursor("ring-dao")
	if err != nil {
		t.Fatalf("GetLastTrackedProposalCursor() error = %v", err)
	}
	if got, want := blockNumber, int64(123); got != want {
		t.Fatalf("block number = %d, want %d", got, want)
	}
	if got, want := storedProposalID, proposalID; got != want {
		t.Fatalf("proposal id length = %d, want preserved length %d", len(got), len(want))
	}
}

func TestRawDaoConfigIncludesRegistryFeatures(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.Exec(`CREATE TABLE dgv_dao (id TEXT PRIMARY KEY, code TEXT UNIQUE NOT NULL, features TEXT)`).Error; err != nil {
		t.Fatalf("create DAO table: %v", err)
	}
	if err := db.Exec(`CREATE TABLE dgv_dao_config (id TEXT PRIMARY KEY, dao_code TEXT UNIQUE NOT NULL, config TEXT NOT NULL)`).Error; err != nil {
		t.Fatalf("create DAO config table: %v", err)
	}
	if err := db.Exec(`INSERT INTO dgv_dao (id, code, features) VALUES (?, ?, ?)`, "dao-id", "demo", `["proposal-comments","proposal-drafts"]`).Error; err != nil {
		t.Fatalf("seed DAO: %v", err)
	}
	if err := db.Exec(`INSERT INTO dgv_dao_config (id, dao_code, config) VALUES (?, ?, ?)`, "config-id", "demo", "code: demo\nname: Demo\n").Error; err != nil {
		t.Fatalf("seed DAO config: %v", err)
	}

	service := &DaoConfigService{db: db}
	content, err := service.RawConfig(gqlmodels.GetDaoConfigInput{DaoCode: "demo"})
	if err != nil {
		t.Fatalf("RawConfig(): %v", err)
	}
	var document struct {
		Features []string `yaml:"features"`
	}
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if got, want := strings.Join(document.Features, ","), "proposal-comments,proposal-drafts"; got != want {
		t.Fatalf("features = %q, want %q", got, want)
	}
}

func TestRawDaoConfigMissingDaoReturnsServiceError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.Exec(`CREATE TABLE dgv_dao (id TEXT PRIMARY KEY, code TEXT UNIQUE NOT NULL, features TEXT)`).Error; err != nil {
		t.Fatalf("create DAO table: %v", err)
	}
	service := &DaoConfigService{db: db}
	_, err = service.featureNames("missing")
	if err == nil || err.Error() != "dao config not found" {
		t.Fatalf("featureNames() error = %v", err)
	}
}

func TestConvertToGqlDaoMapsTagsAndDomains(t *testing.T) {
	t.Parallel()

	service := &DaoService{}
	dao := service.convertToGqlDao(dbmodels.Dao{
		Code:    "ring-dao",
		Tags:    `["governance"]`,
		Domains: `["ringdao.com"]`,
	})

	if got, want := len(dao.Tags), 1; got != want {
		t.Fatalf("len(Tags) = %d, want %d", got, want)
	}
	if got, want := len(dao.Domains), 1; got != want {
		t.Fatalf("len(Domains) = %d, want %d", got, want)
	}
}

func TestApplyDaoConfigOutputOverridesNextModeRewritesIndexerEndpoint(t *testing.T) {
	t.Parallel()

	document := map[string]interface{}{
		"code": "aixbt-dao",
		"indexer": map[string]interface{}{
			"endpoint": "https://indexer.degov.ai/aixbt-dao/graphql",
		},
	}
	applyDaoConfigOutputOverrides(document, "aixbt-dao", "next", "https://indexer.next.degov.ai/{code}/graphql")

	if got, want := getNestedString(document, "indexer", "endpoint"), "https://indexer.next.degov.ai/aixbt-dao/graphql"; got != want {
		t.Fatalf("Indexer.Endpoint = %q, want %q", got, want)
	}
}

func TestApplyDaoConfigOutputOverridesPreservesCanonicalMode(t *testing.T) {
	t.Parallel()

	document := map[string]interface{}{
		"code": "aixbt-dao",
		"indexer": map[string]interface{}{
			"endpoint": "https://indexer.degov.ai/aixbt-dao/graphql",
		},
	}
	applyDaoConfigOutputOverrides(document, "aixbt-dao", "", "https://indexer.next.degov.ai/{code}/graphql")

	if got, want := getNestedString(document, "indexer", "endpoint"), "https://indexer.degov.ai/aixbt-dao/graphql"; got != want {
		t.Fatalf("Indexer.Endpoint = %q, want %q", got, want)
	}
}

func TestRenderDaoConfigJSONIncludesRewrittenEndpoint(t *testing.T) {
	t.Parallel()

	content, err := renderDaoConfig(map[string]interface{}{
		"code": "aixbt-dao",
		"indexer": map[string]interface{}{
			"endpoint": "https://indexer.next.degov.ai/aixbt-dao/graphql",
		},
	}, gqlmodels.ConfigFormatJSON)
	if err != nil {
		t.Fatalf("renderDaoConfig returned error: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(content), &decoded); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}

	indexer, ok := decoded["indexer"].(map[string]interface{})
	if !ok {
		t.Fatalf("decoded indexer = %#v, want object", decoded["indexer"])
	}
	if got, want := indexer["endpoint"], "https://indexer.next.degov.ai/aixbt-dao/graphql"; got != want {
		t.Fatalf("decoded indexer.endpoint = %#v, want %q", got, want)
	}
}

func TestRenderDaoConfigJSONPreservesUnknownFields(t *testing.T) {
	t.Parallel()

	content, err := renderDaoConfig(map[string]interface{}{
		"code": "aixbt-dao",
		"futureField": map[string]interface{}{
			"enabled": true,
		},
	}, gqlmodels.ConfigFormatJSON)
	if err != nil {
		t.Fatalf("renderDaoConfig returned error: %v", err)
	}

	if !strings.Contains(content, "\"futureField\"") {
		t.Fatalf("rendered JSON %q does not preserve unknown fields", content)
	}
}

func TestRenderDaoConfigYAMLQuotesAddressStrings(t *testing.T) {
	t.Parallel()

	content, err := renderDaoConfig(map[string]interface{}{
		"chain": map[string]interface{}{
			"id": 1,
		},
		"contracts": map[string]interface{}{
			"governor": "0x7ae22bebF28366c328d5558E6Fad935487299DfE",
			"governorToken": map[string]interface{}{
				"address":  "0x970C30646E5c95DC77A3D768C4362E113Ed92b5b",
				"standard": "ERC20",
			},
			"timeLock": "0xEd4f981249Dde7Cd3c295fc28CB934D4682d7ef9",
		},
	}, gqlmodels.ConfigFormatYaml)
	if err != nil {
		t.Fatalf("renderDaoConfig returned error: %v", err)
	}

	if !strings.Contains(content, `governor: "0x7ae22bebF28366c328d5558E6Fad935487299DfE"`) {
		t.Fatalf("rendered YAML %q does not quote governor address", content)
	}
	if !strings.Contains(content, `address: "0x970C30646E5c95DC77A3D768C4362E113Ed92b5b"`) {
		t.Fatalf("rendered YAML %q does not quote governorToken.address", content)
	}
	if !strings.Contains(content, `timeLock: "0xEd4f981249Dde7Cd3c295fc28CB934D4682d7ef9"`) {
		t.Fatalf("rendered YAML %q does not quote timeLock", content)
	}
	if !strings.Contains(content, "id: 1") {
		t.Fatalf("rendered YAML %q does not preserve numeric fields", content)
	}
}
