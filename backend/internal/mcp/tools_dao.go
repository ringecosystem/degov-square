package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	dbmodels "github.com/ringecosystem/degov-square/database/models"
	gqlmodels "github.com/ringecosystem/degov-square/graph/models"
	"github.com/ringecosystem/degov-square/types"
)

var daoCodePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$`)

func addDaoTools(server *sdkmcp.Server, cfg Config) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "list_daos",
		Title:       "List DAOs",
		Description: "Return a bounded list of public DAO summaries.",
		Annotations: readOnlyToolAnnotations(),
		InputSchema: listDaosInputSchema(),
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input listDaosInput) (*sdkmcp.CallToolResult, listDaosOutput, error) {
		serviceInput, err := toListDaosServiceInput(input)
		if err != nil {
			return nil, listDaosOutput{}, err
		}
		daos, err := cfg.DaoService.ListDaos(types.BasicInput[*types.ListDaosInput]{
			Input: serviceInput,
		})
		if err != nil {
			return nil, listDaosOutput{}, err
		}

		limit := normalizeListDaosLimit(input.Limit)
		if len(daos) > limit {
			daos = daos[:limit]
		}

		output := listDaosOutput{
			Daos:  make([]daoSummaryOutput, 0, len(daos)),
			Count: len(daos),
			Limit: limit,
		}
		for _, dao := range daos {
			if dao == nil {
				continue
			}
			output.Daos = append(output.Daos, daoSummaryFromGQL(dao))
		}
		output.Count = len(output.Daos)

		return nil, output, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "get_dao",
		Title:       "Get DAO",
		Description: "Return public DAO metadata for one DAO code.",
		Annotations: readOnlyToolAnnotations(),
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input getDaoInput) (*sdkmcp.CallToolResult, daoDetailOutput, error) {
		daoCode, err := normalizeDaoCode(input.DaoCode)
		if err != nil {
			return nil, daoDetailOutput{}, err
		}

		dao, err := cfg.DaoService.Inspect(types.BasicInput[string]{Input: daoCode})
		if err != nil {
			return nil, daoDetailOutput{}, daoToolError("DAO", daoCode, err)
		}
		if dao == nil {
			return nil, daoDetailOutput{}, notFoundError("DAO", daoCode)
		}

		return nil, daoDetailFromGQL(dao), nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "get_dao_config",
		Title:       "Get DAO Config",
		Description: "Return the public DAO registry config for one DAO code.",
		Annotations: readOnlyToolAnnotations(),
		InputSchema: getDaoConfigInputSchema(),
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input getDaoConfigInput) (*sdkmcp.CallToolResult, daoConfigOutput, error) {
		daoCode, err := normalizeDaoCode(input.DaoCode)
		if err != nil {
			return nil, daoConfigOutput{}, err
		}

		format, formatName, err := normalizeConfigFormat(input.Format)
		if err != nil {
			return nil, daoConfigOutput{}, err
		}

		content, err := cfg.DaoConfigService.RawConfig(gqlmodels.GetDaoConfigInput{
			DaoCode: daoCode,
			Format:  &format,
		})
		if err != nil {
			return nil, daoConfigOutput{}, daoToolError("DAO config", daoCode, err)
		}

		return nil, daoConfigOutput{
			DaoCode: daoCode,
			Format:  formatName,
			Content: content,
		}, nil
	})
}

func listDaosInputSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"codes": {
				Type:        "array",
				Description: "Optional DAO codes to filter by, for example ring-dao.",
				Items: &jsonschema.Schema{
					Type:    "string",
					Pattern: daoCodePattern.String(),
				},
				MaxItems: jsonschema.Ptr(20),
			},
			"state": {
				Type:        "string",
				Description: "Optional DAO state to filter by.",
				Enum: []any{
					string(dbmodels.DaoStateActive),
					string(dbmodels.DaoStateDraft),
					string(dbmodels.DaoStateInactive),
					strings.ToLower(string(dbmodels.DaoStateActive)),
					strings.ToLower(string(dbmodels.DaoStateDraft)),
					strings.ToLower(string(dbmodels.DaoStateInactive)),
				},
			},
			"limit": {
				Type:        "integer",
				Description: "Maximum number of DAOs to return. Values above 100 are capped.",
			},
		},
	}
}

func getDaoConfigInputSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:     "object",
		Required: []string{"daoCode"},
		Properties: map[string]*jsonschema.Schema{
			"daoCode": {
				Type:        "string",
				Description: "DAO code, for example ring-dao.",
				Pattern:     daoCodePattern.String(),
			},
			"format": {
				Type:        "string",
				Description: "Config format. Defaults to json.",
				Enum:        []any{"json", "yaml"},
				Default:     json.RawMessage(`"json"`),
			},
		},
	}
}

func normalizeListDaosLimit(limit int) int {
	if limit <= 0 {
		return defaultListDaosLimit
	}
	if limit > maxListDaosLimit {
		return maxListDaosLimit
	}
	return limit
}

func toListDaosServiceInput(input listDaosInput) (*types.ListDaosInput, error) {
	serviceInput := &types.ListDaosInput{}
	if len(input.Codes) > 0 {
		codes := make([]string, 0, len(input.Codes))
		for _, code := range input.Codes {
			normalized, err := normalizeDaoCode(code)
			if err != nil {
				return nil, err
			}
			codes = append(codes, normalized)
		}
		serviceInput.Codes = &codes
	}
	if strings.TrimSpace(input.State) != "" {
		normalized := dbmodels.DaoState(strings.ToUpper(strings.TrimSpace(input.State)))
		switch normalized {
		case dbmodels.DaoStateActive, dbmodels.DaoStateDraft, dbmodels.DaoStateInactive:
			states := []dbmodels.DaoState{normalized}
			serviceInput.State = &states
		default:
			return nil, invalidParamsError("state must be ACTIVE, DRAFT, or INACTIVE")
		}
	}
	return serviceInput, nil
}

func normalizeDaoCode(daoCode string) (string, error) {
	normalized := strings.TrimSpace(daoCode)
	if normalized == "" {
		return "", invalidParamsError("daoCode is required")
	}
	if !daoCodePattern.MatchString(normalized) {
		return "", invalidParamsError("daoCode contains invalid characters")
	}
	return normalized, nil
}

func normalizeConfigFormat(format string) (gqlmodels.ConfigFormat, string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "json":
		return gqlmodels.ConfigFormatJSON, "json", nil
	case "yaml", "yml":
		return gqlmodels.ConfigFormatYaml, "yaml", nil
	default:
		return "", "", invalidParamsError("format must be json or yaml")
	}
}

func daoSummaryFromGQL(dao *gqlmodels.Dao) daoSummaryOutput {
	return daoSummaryOutput{
		DaoCode:               dao.Code,
		Name:                  dao.Name,
		Logo:                  dao.Logo,
		Endpoint:              dao.Endpoint,
		ChainID:               dao.ChainID,
		ChainName:             dao.ChainName,
		ChainLogo:             dao.ChainLogo,
		State:                 dao.State,
		Domains:               dao.Domains,
		Tags:                  dao.Tags,
		MetricsCountProposals: dao.MetricsCountProposals,
		MetricsCountMembers:   dao.MetricsCountMembers,
		MetricsSumPower:       dao.MetricsSumPower,
		MetricsCountVote:      dao.MetricsCountVote,
	}
}

func daoDetailFromGQL(dao *gqlmodels.Dao) daoDetailOutput {
	chips := make([]daoChipOutput, 0, len(dao.Chips))
	for _, chip := range dao.Chips {
		if chip == nil {
			continue
		}
		chips = append(chips, daoChipOutput{
			ChipCode: chip.ChipCode,
			Flag:     chip.Flag,
		})
	}

	summary := daoSummaryFromGQL(dao)
	return daoDetailOutput{
		DaoCode:               summary.DaoCode,
		Name:                  summary.Name,
		Logo:                  summary.Logo,
		Endpoint:              summary.Endpoint,
		ChainID:               summary.ChainID,
		ChainName:             summary.ChainName,
		ChainLogo:             summary.ChainLogo,
		State:                 summary.State,
		Domains:               summary.Domains,
		Tags:                  summary.Tags,
		MetricsCountProposals: summary.MetricsCountProposals,
		MetricsCountMembers:   summary.MetricsCountMembers,
		MetricsSumPower:       summary.MetricsSumPower,
		MetricsCountVote:      summary.MetricsCountVote,
		Chips:                 chips,
	}
}

func daoToolError(resource, daoCode string, err error) error {
	if isNotFoundError(err) {
		return notFoundError(resource, daoCode)
	}
	return err
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "not found") || strings.Contains(message, "record not found")
}

func invalidParamsError(message string) error {
	return &jsonrpc.Error{
		Code:    jsonrpc.CodeInvalidParams,
		Message: message,
	}
}

func notFoundError(resource, daoCode string) error {
	return &jsonrpc.Error{
		Code:    jsonrpc.CodeInvalidParams,
		Message: fmt.Sprintf("%s %q not found", resource, daoCode),
	}
}
