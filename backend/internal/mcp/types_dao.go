package mcp

import (
	gqlmodels "github.com/ringecosystem/degov-square/graph/models"
	"github.com/ringecosystem/degov-square/types"
)

const (
	defaultListDaosLimit = 50
	maxListDaosLimit     = 100
)

type daoService interface {
	ListDaos(types.BasicInput[*types.ListDaosInput]) ([]*gqlmodels.Dao, error)
	Inspect(types.BasicInput[string]) (*gqlmodels.Dao, error)
}

type daoConfigService interface {
	RawConfig(gqlmodels.GetDaoConfigInput) (string, error)
}

type listDaosInput struct {
	Codes []string `json:"codes,omitempty" jsonschema:"DAO codes to filter by."`
	State []string `json:"state,omitempty" jsonschema:"DAO states to filter by."`
	Limit int      `json:"limit,omitempty" jsonschema:"Maximum number of DAOs to return. Values above 100 are capped."`
}

type getDaoInput struct {
	DaoCode string `json:"daoCode" jsonschema:"DAO code."`
}

type getDaoConfigInput struct {
	DaoCode string `json:"daoCode" jsonschema:"DAO code."`
	Format  string `json:"format,omitempty" jsonschema:"Config format: json or yaml. Defaults to json."`
}

type listDaosOutput struct {
	Daos  []daoSummaryOutput `json:"daos"`
	Count int                `json:"count"`
	Limit int                `json:"limit"`
}

type daoSummaryOutput struct {
	DaoCode               string   `json:"daoCode"`
	Name                  string   `json:"name"`
	Logo                  *string  `json:"logo,omitempty"`
	Endpoint              string   `json:"endpoint"`
	ChainID               int32    `json:"chainId"`
	ChainName             string   `json:"chainName"`
	ChainLogo             *string  `json:"chainLogo,omitempty"`
	State                 string   `json:"state"`
	Domains               []string `json:"domains,omitempty"`
	Tags                  []string `json:"tags,omitempty"`
	MetricsCountProposals int32    `json:"metricsCountProposals"`
	MetricsCountMembers   int32    `json:"metricsCountMembers"`
	MetricsSumPower       string   `json:"metricsSumPower"`
	MetricsCountVote      int32    `json:"metricsCountVote"`
}

type daoDetailOutput struct {
	DaoCode               string          `json:"daoCode"`
	Name                  string          `json:"name"`
	Logo                  *string         `json:"logo,omitempty"`
	Endpoint              string          `json:"endpoint"`
	ChainID               int32           `json:"chainId"`
	ChainName             string          `json:"chainName"`
	ChainLogo             *string         `json:"chainLogo,omitempty"`
	State                 string          `json:"state"`
	Domains               []string        `json:"domains,omitempty"`
	Tags                  []string        `json:"tags,omitempty"`
	MetricsCountProposals int32           `json:"metricsCountProposals"`
	MetricsCountMembers   int32           `json:"metricsCountMembers"`
	MetricsSumPower       string          `json:"metricsSumPower"`
	MetricsCountVote      int32           `json:"metricsCountVote"`
	Chips                 []daoChipOutput `json:"chips,omitempty"`
}

type daoChipOutput struct {
	ChipCode string `json:"chipCode"`
	Flag     string `json:"flag,omitempty"`
}

type daoConfigOutput struct {
	DaoCode string `json:"daoCode"`
	Format  string `json:"format"`
	Content string `json:"content"`
}
