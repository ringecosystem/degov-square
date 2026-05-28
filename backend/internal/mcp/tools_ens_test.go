package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ringecosystem/degov-square/services"
)

type fakeENSService struct {
	resolveRecord  *services.ENSRecord
	resolveRecords *services.ENSPublicRecords
	resolveErr     error
	recordsErr     error
	delay          time.Duration
	resolveCalls   int
	recordsCalls   int
	daoCode        *string
	address        *string
	name           *string
	recordsName    string
}

func (s *fakeENSService) Resolve(ctx context.Context, daoCode *string, address *string, name *string) (*services.ENSRecord, error) {
	s.resolveCalls++
	s.daoCode = daoCode
	s.address = address
	s.name = name
	if s.delay > 0 {
		select {
		case <-time.After(s.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return s.resolveRecord, s.resolveErr
}

func (s *fakeENSService) ResolveRecords(ctx context.Context, name string) (*services.ENSPublicRecords, error) {
	s.recordsCalls++
	s.recordsName = name
	if s.delay > 0 {
		select {
		case <-time.After(s.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return s.resolveRecords, s.recordsErr
}

func TestResolveENSToolResolvesName(t *testing.T) {
	address := "0x0000000000000000000000000000000000000001"
	service := &fakeENSService{
		resolveRecord: &services.ENSRecord{
			Name:    stringPtr("alice.eth"),
			Address: &address,
		},
	}
	server := NewServer(Config{Name: "degov-square", Version: "test-version", ENSService: service})

	result := callProposalTool(t, server, "resolve_ens", map[string]any{"name": "Alice.ETH"})
	content := requireStructuredContent(t, result)

	if got, want := content["name"], "alice.eth"; got != want {
		t.Fatalf("name = %v, want %v", got, want)
	}
	if got, want := content["address"], address; got != want {
		t.Fatalf("address = %v, want %v", got, want)
	}
	if service.name == nil || *service.name != "alice.eth" {
		t.Fatalf("service name = %#v, want alice.eth", service.name)
	}
	if service.address != nil {
		t.Fatalf("service address = %#v, want nil", service.address)
	}
}

func TestResolveENSToolResolvesAddress(t *testing.T) {
	name := "alice.eth"
	service := &fakeENSService{
		resolveRecord: &services.ENSRecord{
			Name:    &name,
			Address: stringPtr("0x0000000000000000000000000000000000000001"),
		},
	}
	server := NewServer(Config{Name: "degov-square", Version: "test-version", ENSService: service})

	result := callProposalTool(t, server, "resolve_ens", map[string]any{
		"address": "0x0000000000000000000000000000000000000001",
	})
	content := requireStructuredContent(t, result)

	if got, want := content["name"], name; got != want {
		t.Fatalf("name = %v, want %v", got, want)
	}
	if service.address == nil || *service.address != "0x0000000000000000000000000000000000000001" {
		t.Fatalf("service address = %#v, want normalized address", service.address)
	}
	if service.name != nil {
		t.Fatalf("service name = %#v, want nil", service.name)
	}
}

func TestResolveENSRecordsToolReturnsPublicRecords(t *testing.T) {
	service := &fakeENSService{
		resolveRecords: &services.ENSPublicRecords{
			Name:        "alice.eth",
			Address:     stringPtr("0x0000000000000000000000000000000000000001"),
			Contenthash: stringPtr("ipfs://bafybeigdyrzt"),
			Text: map[string]string{
				"avatar": "eip155:1/erc721:0xabc/1",
				"url":    "https://alice.example",
			},
		},
	}
	server := NewServer(Config{Name: "degov-square", Version: "test-version", ENSService: service})

	result := callProposalTool(t, server, "resolve_ens_records", map[string]any{"name": "Alice.ETH"})
	content := requireStructuredContent(t, result)

	if got, want := content["name"], "alice.eth"; got != want {
		t.Fatalf("name = %v, want %v", got, want)
	}
	if got, want := content["address"], "0x0000000000000000000000000000000000000001"; got != want {
		t.Fatalf("address = %v, want %v", got, want)
	}
	text := content["text"].(map[string]any)
	if got, want := text["url"], "https://alice.example"; got != want {
		t.Fatalf("url text = %v, want %v", got, want)
	}
	if got, want := service.recordsName, "alice.eth"; got != want {
		t.Fatalf("records name = %q, want %q", got, want)
	}
}

func TestENSToolsRejectInvalidInputBeforeServiceCall(t *testing.T) {
	tests := []struct {
		name      string
		tool      string
		arguments map[string]any
		wantError string
	}{
		{
			name:      "resolve requires one query",
			tool:      "resolve_ens",
			arguments: map[string]any{},
			wantError: "invalid_ens_query",
		},
		{
			name:      "resolve rejects both query types",
			tool:      "resolve_ens",
			arguments: map[string]any{"name": "alice.eth", "address": "0x0000000000000000000000000000000000000001"},
			wantError: "invalid_ens_query",
		},
		{
			name:      "resolve rejects invalid address",
			tool:      "resolve_ens",
			arguments: map[string]any{"address": "0x1234"},
			wantError: "invalid_ens_address",
		},
		{
			name:      "resolve rejects invalid name",
			tool:      "resolve_ens",
			arguments: map[string]any{"name": "-bad.eth"},
			wantError: "invalid_ens_name",
		},
		{
			name:      "records rejects invalid name",
			tool:      "resolve_ens_records",
			arguments: map[string]any{"name": "bad name.eth"},
			wantError: "invalid_ens_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeENSService{}
			server := NewServer(Config{Name: "degov-square", Version: "test-version", ENSService: service})

			result := callProposalTool(t, server, tt.tool, tt.arguments)

			requireToolErrorContains(t, result, tt.wantError)
			if service.resolveCalls != 0 || service.recordsCalls != 0 {
				t.Fatalf("service calls = resolve:%d records:%d, want none", service.resolveCalls, service.recordsCalls)
			}
		})
	}
}

func TestENSToolsReturnStructuredFailures(t *testing.T) {
	tests := []struct {
		name      string
		tool      string
		arguments map[string]any
		service   *fakeENSService
		wantError string
	}{
		{
			name:      "resolve not found",
			tool:      "resolve_ens",
			arguments: map[string]any{"name": "missing.eth"},
			service:   &fakeENSService{resolveRecord: &services.ENSRecord{Name: stringPtr("missing.eth")}},
			wantError: "ens_not_found",
		},
		{
			name:      "resolve service failure",
			tool:      "resolve_ens",
			arguments: map[string]any{"name": "alice.eth"},
			service:   &fakeENSService{resolveErr: errors.New("rpc failed")},
			wantError: "ens_resolution_failed",
		},
		{
			name:      "records not found",
			tool:      "resolve_ens_records",
			arguments: map[string]any{"name": "missing.eth"},
			service:   &fakeENSService{resolveRecords: &services.ENSPublicRecords{Name: "missing.eth"}},
			wantError: "ens_records_not_found",
		},
		{
			name:      "records service failure",
			tool:      "resolve_ens_records",
			arguments: map[string]any{"name": "alice.eth"},
			service:   &fakeENSService{recordsErr: errors.New("rpc failed")},
			wantError: "ens_records_failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(Config{Name: "degov-square", Version: "test-version", ENSService: tt.service})

			result := callProposalTool(t, server, tt.tool, tt.arguments)

			requireToolErrorContains(t, result, tt.wantError)
		})
	}
}

func TestENSToolsBoundExternalCallsByTimeout(t *testing.T) {
	server := NewServer(Config{
		Name:              "degov-square",
		Version:           "test-version",
		ENSService:        &fakeENSService{delay: 100 * time.Millisecond},
		ENSResolveTimeout: 10 * time.Millisecond,
	})

	started := time.Now()
	result := callProposalTool(t, server, "resolve_ens", map[string]any{"name": "alice.eth"})

	requireToolErrorContains(t, result, "ens_resolution_timeout")
	if elapsed := time.Since(started); elapsed > 80*time.Millisecond {
		t.Fatalf("resolve_ens elapsed = %s, want timeout before service delay", elapsed)
	}
}
