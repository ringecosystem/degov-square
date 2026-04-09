package services

import (
	"strings"
	"testing"

	dbmodels "github.com/ringecosystem/degov-square/database/models"
	gqlmodels "github.com/ringecosystem/degov-square/graph/models"
	"github.com/ringecosystem/degov-square/types"
)

func TestConvertToGqlDaoMapsOffsetTrackingProposal(t *testing.T) {
	t.Parallel()

	service := &DaoService{}
	dao := service.convertToGqlDao(dbmodels.Dao{
		Code:                "ring-dao",
		OffsetTrackingBlock: 42,
		Tags:                `["governance"]`,
		Domains:             `["ringdao.com"]`,
	})

	if got, want := dao.OffsetTrackingProposal, int32(42); got != want {
		t.Fatalf("OffsetTrackingProposal = %d, want %d", got, want)
	}
	if got, want := len(dao.Tags), 1; got != want {
		t.Fatalf("len(Tags) = %d, want %d", got, want)
	}
	if got, want := len(dao.Domains), 1; got != want {
		t.Fatalf("len(Domains) = %d, want %d", got, want)
	}
}

func TestApplyDaoConfigOutputOverridesNextModeRewritesIndexerEndpoint(t *testing.T) {
	t.Parallel()

	daoConfig := applyDaoConfigOutputOverrides(types.DaoConfig{
		Code: "aixbt-dao",
		Indexer: struct {
			Endpoint   string `yaml:"endpoint"`
			StartBlock int    `yaml:"startBlock"`
			RPC        string `yaml:"rpc"`
			Gateway    string `yaml:"gateway"`
		}{
			Endpoint: "https://indexer.degov.ai/aixbt-dao/graphql",
		},
	}, "aixbt-dao", "next", "https://indexer.next.degov.ai/{code}/graphql")

	if got, want := daoConfig.Indexer.Endpoint, "https://indexer.next.degov.ai/aixbt-dao/graphql"; got != want {
		t.Fatalf("Indexer.Endpoint = %q, want %q", got, want)
	}
}

func TestApplyDaoConfigOutputOverridesPreservesCanonicalMode(t *testing.T) {
	t.Parallel()

	daoConfig := applyDaoConfigOutputOverrides(types.DaoConfig{
		Code: "aixbt-dao",
		Indexer: struct {
			Endpoint   string `yaml:"endpoint"`
			StartBlock int    `yaml:"startBlock"`
			RPC        string `yaml:"rpc"`
			Gateway    string `yaml:"gateway"`
		}{
			Endpoint: "https://indexer.degov.ai/aixbt-dao/graphql",
		},
	}, "aixbt-dao", "", "https://indexer.next.degov.ai/{code}/graphql")

	if got, want := daoConfig.Indexer.Endpoint, "https://indexer.degov.ai/aixbt-dao/graphql"; got != want {
		t.Fatalf("Indexer.Endpoint = %q, want %q", got, want)
	}
}

func TestRenderDaoConfigJSONIncludesRewrittenEndpoint(t *testing.T) {
	t.Parallel()

	content, err := renderDaoConfig(types.DaoConfig{
		Code: "aixbt-dao",
		Indexer: struct {
			Endpoint   string `yaml:"endpoint"`
			StartBlock int    `yaml:"startBlock"`
			RPC        string `yaml:"rpc"`
			Gateway    string `yaml:"gateway"`
		}{
			Endpoint: "https://indexer.next.degov.ai/aixbt-dao/graphql",
		},
	}, gqlmodels.ConfigFormatJSON)
	if err != nil {
		t.Fatalf("renderDaoConfig returned error: %v", err)
	}

	if want := "\"endpoint\": \"https://indexer.next.degov.ai/aixbt-dao/graphql\""; !strings.Contains(content, want) {
		t.Fatalf("rendered JSON %q does not contain %q", content, want)
	}
}
