package services

import (
	"encoding/json"
	"strings"
	"testing"

	dbmodels "github.com/ringecosystem/degov-square/database/models"
	gqlmodels "github.com/ringecosystem/degov-square/graph/models"
)

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
