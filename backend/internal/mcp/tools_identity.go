package mcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ringecosystem/degov-square/services"
)

const maxAddressIdentityBatch = 50

func hydrateAddressIdentities(ctx context.Context, cfg Config, daoCode string, addresses []string) map[string]addressIdentityOutput {
	output := make(map[string]addressIdentityOutput)
	if len(addresses) == 0 {
		return output
	}

	cfg = withDefaultENSServices(cfg)
	normalizedDaoCode := daoCode
	seen := make(map[string]struct{})
	for _, rawAddress := range addresses {
		address, err := normalizeENSAddress(rawAddress)
		if err != nil {
			continue
		}
		if _, ok := seen[address]; ok {
			continue
		}
		seen[address] = struct{}{}
		output[address] = addressIdentityOutput{Address: address}
		if len(seen) >= maxAddressIdentityBatch {
			break
		}
	}

	for address := range output {
		addressValue := address
		record, err := resolveENSWithTimeout(ctx, cfg, &normalizedDaoCode, &addressValue, nil)
		if err != nil {
			slog.Debug("MCP ENS identity enrichment failed", "daoCode", daoCode, "address", address, "err", err)
			continue
		}
		if record == nil || record.Name == nil || *record.Name == "" {
			continue
		}
		identity := output[address]
		identity.ENSName = record.Name
		output[address] = identity
	}

	return output
}

func addressIdentityFromMap(identities map[string]addressIdentityOutput, address string) addressIdentityOutput {
	lower, err := normalizeENSAddress(address)
	if err != nil {
		return addressIdentityOutput{Address: address}
	}
	if identity, ok := identities[lower]; ok {
		return identity
	}
	return addressIdentityOutput{Address: lower}
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

func normalizeENSAddress(address string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(address))
	if !common.IsHexAddress(normalized) {
		return "", errors.New("invalid_ens_address: address must be a valid EVM address")
	}
	return normalized, nil
}
