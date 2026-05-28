package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ringecosystem/degov-square/services"
	"github.com/wealdtech/go-ens/v3"
)

func addENSTools(server *sdkmcp.Server, cfg Config) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "resolve_ens",
		Title:       "Resolve ENS",
		Description: "Resolve an ENS name to an address or reverse-resolve an EVM address to an ENS name.",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input resolveENSInput) (*sdkmcp.CallToolResult, resolveENSOutput, error) {
		return resolveENSTool(ctx, cfg, input)
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "resolve_ens_records",
		Title:       "Resolve ENS Records",
		Description: "Return public records for an ENS name.",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input resolveENSRecordsInput) (*sdkmcp.CallToolResult, resolveENSRecordsOutput, error) {
		return resolveENSRecordsTool(ctx, cfg, input)
	})
}

func resolveENSTool(ctx context.Context, cfg Config, input resolveENSInput) (*sdkmcp.CallToolResult, resolveENSOutput, error) {
	name := strings.TrimSpace(input.Name)
	address := strings.TrimSpace(input.Address)
	if (name == "" && address == "") || (name != "" && address != "") {
		return nil, resolveENSOutput{}, errors.New("invalid_ens_query: exactly one of name or address is required")
	}

	var normalizedName *string
	var normalizedAddress *string
	if name != "" {
		value, err := normalizeENSName(name)
		if err != nil {
			return nil, resolveENSOutput{}, err
		}
		normalizedName = &value
	}
	if address != "" {
		value, err := normalizeENSAddress(address)
		if err != nil {
			return nil, resolveENSOutput{}, err
		}
		normalizedAddress = &value
	}

	record, err := resolveENSWithTimeout(ctx, cfg, nil, normalizedAddress, normalizedName)
	if err != nil {
		return nil, resolveENSOutput{}, err
	}
	if record == nil || (record.Name == nil && record.Address == nil) {
		return nil, resolveENSOutput{}, errors.New("ens_not_found: ENS resolution was not found")
	}
	if normalizedName != nil && record.Address == nil {
		return nil, resolveENSOutput{}, fmt.Errorf("ens_not_found: ENS name %q was not found", *normalizedName)
	}
	if normalizedAddress != nil && record.Name == nil {
		return nil, resolveENSOutput{}, fmt.Errorf("ens_not_found: ENS address %q was not found", *normalizedAddress)
	}

	return nil, ensRecordOutput(record), nil
}

func resolveENSRecordsTool(ctx context.Context, cfg Config, input resolveENSRecordsInput) (*sdkmcp.CallToolResult, resolveENSRecordsOutput, error) {
	name, err := normalizeENSName(input.Name)
	if err != nil {
		return nil, resolveENSRecordsOutput{}, err
	}

	records, err := resolveENSRecordsWithTimeout(ctx, cfg, name)
	if err != nil {
		return nil, resolveENSRecordsOutput{}, err
	}
	if records == nil || (records.Address == nil && records.Contenthash == nil && len(records.Text) == 0) {
		return nil, resolveENSRecordsOutput{}, fmt.Errorf("ens_records_not_found: ENS records for %q were not found", name)
	}

	return nil, ensRecordsOutput(records), nil
}

func resolveENSWithTimeout(ctx context.Context, cfg Config, daoCode *string, address *string, name *string) (*services.ENSRecord, error) {
	timeout := cfg.ENSResolveTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	type result struct {
		record *services.ENSRecord
		err    error
	}
	resultCh := make(chan result, 1)
	go func() {
		record, err := cfg.ENSService.Resolve(ctx, daoCode, address, name)
		resultCh <- result{record: record, err: err}
	}()

	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("ens_resolution_timeout: exceeded %s", timeout)
		}
		return nil, fmt.Errorf("ens_resolution_cancelled: %w", ctx.Err())
	case result := <-resultCh:
		if result.err != nil {
			return nil, fmt.Errorf("ens_resolution_failed: %w", result.err)
		}
		return result.record, nil
	}
}

func resolveENSRecordsWithTimeout(ctx context.Context, cfg Config, name string) (*services.ENSPublicRecords, error) {
	timeout := cfg.ENSResolveTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	type result struct {
		records *services.ENSPublicRecords
		err     error
	}
	resultCh := make(chan result, 1)
	go func() {
		records, err := cfg.ENSService.ResolveRecords(ctx, name)
		resultCh <- result{records: records, err: err}
	}()

	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("ens_records_timeout: exceeded %s", timeout)
		}
		return nil, fmt.Errorf("ens_records_cancelled: %w", ctx.Err())
	case result := <-resultCh:
		if result.err != nil {
			return nil, fmt.Errorf("ens_records_failed: %w", result.err)
		}
		return result.records, nil
	}
}

func normalizeENSAddress(address string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(address))
	if !common.IsHexAddress(normalized) {
		return "", errors.New("invalid_ens_address: address must be a valid EVM address")
	}
	return normalized, nil
}

func normalizeENSName(name string) (string, error) {
	normalized, err := ens.NormaliseDomainStrict(strings.TrimSpace(name))
	if err != nil {
		return "", fmt.Errorf("invalid_ens_name: %w", err)
	}
	if normalized == "" || !strings.Contains(normalized, ".") || strings.HasPrefix(normalized, ".") || strings.HasSuffix(normalized, ".") {
		return "", errors.New("invalid_ens_name: name must be a fully-qualified ENS name")
	}
	labels := strings.Split(normalized, ".")
	for _, label := range labels {
		if label == "" || label == "*" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return "", errors.New("invalid_ens_name: name contains invalid labels")
		}
	}
	return normalized, nil
}

func ensRecordOutput(record *services.ENSRecord) resolveENSOutput {
	return resolveENSOutput{
		Name:    record.Name,
		Address: record.Address,
	}
}

func ensRecordsOutput(records *services.ENSPublicRecords) resolveENSRecordsOutput {
	return resolveENSRecordsOutput{
		Name:        records.Name,
		Address:     records.Address,
		Contenthash: records.Contenthash,
		Text:        records.Text,
	}
}
