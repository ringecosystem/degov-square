package services

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestENSService(t *testing.T) *ENSService {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	if err := db.Exec(`
		CREATE TABLE dgv_dao_config (
			id TEXT PRIMARY KEY,
			dao_code TEXT NOT NULL UNIQUE,
			config TEXT NOT NULL
		)
	`).Error; err != nil {
		t.Fatalf("create dao config table: %v", err)
	}

	return &ENSService{
		db:    db,
		cache: make(map[string]ensCacheEntry),
	}
}

func TestENSResolveUsesConfiguredRPCBeforeDaoConfigRPCAndCaches(t *testing.T) {
	service := newTestENSService(t)
	t.Setenv("RPC_URL_1", "https://env-rpc.example")
	daoCode := "demo"
	address := "0x0000000000000000000000000000000000000001"

	if err := service.db.Exec(`
		INSERT INTO dgv_dao_config (id, dao_code, config)
		VALUES (?, ?, ?)
	`, "cfg-1", daoCode, `
code: demo
chain:
  id: 1
  rpcs:
    - https://dao-rpc.example
`).Error; err != nil {
		t.Fatalf("seed dao config: %v", err)
	}

	originalLookup := resolveENSNameViaRPC
	t.Cleanup(func() {
		resolveENSNameViaRPC = originalLookup
	})

	calls := make([]string, 0, 2)
	resolveENSNameViaRPC = func(ctx context.Context, rpcURL string, address string) (*string, error) {
		calls = append(calls, rpcURL)
		if rpcURL == "https://env-rpc.example" {
			return nil, errors.New("temporary failure")
		}
		ensName := "alice.eth"
		return &ensName, nil
	}

	first, err := service.Resolve(context.Background(), &daoCode, &address, nil)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if first == nil || first.Name == nil || *first.Name != "alice.eth" {
		t.Fatalf("expected alice.eth, got %#v", first)
	}

	second, err := service.Resolve(context.Background(), &daoCode, &address, nil)
	if err != nil {
		t.Fatalf("second Resolve returned error: %v", err)
	}
	if second == nil || second.Name == nil || *second.Name != "alice.eth" {
		t.Fatalf("expected cached alice.eth, got %#v", second)
	}

	expectedCalls := []string{"https://env-rpc.example", "https://dao-rpc.example"}
	if !reflect.DeepEqual(calls, expectedCalls) {
		t.Fatalf("expected lookup calls %v, got %v", expectedCalls, calls)
	}
}

func TestENSResolveNameToAddressUsesConfiguredRPCBeforeDaoConfigRPCAndCaches(t *testing.T) {
	service := newTestENSService(t)
	t.Setenv("RPC_URL_1", "https://env-rpc.example")
	daoCode := "demo"
	name := "alice.eth"

	if err := service.db.Exec(`
		INSERT INTO dgv_dao_config (id, dao_code, config)
		VALUES (?, ?, ?)
	`, "cfg-1", daoCode, `
code: demo
chain:
  id: 1
  rpcs:
    - https://dao-rpc.example
`).Error; err != nil {
		t.Fatalf("seed dao config: %v", err)
	}

	originalLookup := resolveENSAddressViaRPC
	t.Cleanup(func() {
		resolveENSAddressViaRPC = originalLookup
	})

	calls := make([]string, 0, 2)
	resolveENSAddressViaRPC = func(ctx context.Context, rpcURL string, name string) (*string, error) {
		calls = append(calls, rpcURL)
		if rpcURL == "https://env-rpc.example" {
			return nil, errors.New("temporary failure")
		}
		address := "0x0000000000000000000000000000000000000001"
		return &address, nil
	}

	first, err := service.Resolve(context.Background(), &daoCode, nil, &name)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if first == nil || first.Address == nil || *first.Address != "0x0000000000000000000000000000000000000001" {
		t.Fatalf("expected resolved address, got %#v", first)
	}

	second, err := service.Resolve(context.Background(), &daoCode, nil, &name)
	if err != nil {
		t.Fatalf("second Resolve returned error: %v", err)
	}
	if second == nil || second.Address == nil || *second.Address != "0x0000000000000000000000000000000000000001" {
		t.Fatalf("expected cached resolved address, got %#v", second)
	}

	expectedCalls := []string{"https://env-rpc.example", "https://dao-rpc.example"}
	if !reflect.DeepEqual(calls, expectedCalls) {
		t.Fatalf("expected lookup calls %v, got %v", expectedCalls, calls)
	}
}

func TestENSCacheExpires(t *testing.T) {
	service := newTestENSService(t)
	t.Setenv("RPC_URL_1", "https://env-rpc.example")
	t.Setenv("DEGOV_ENS_CACHE_TTL", "10ms")
	address := "0x0000000000000000000000000000000000000002"

	originalLookup := resolveENSNameViaRPC
	t.Cleanup(func() {
		resolveENSNameViaRPC = originalLookup
	})

	calls := 0
	resolveENSNameViaRPC = func(ctx context.Context, rpcURL string, address string) (*string, error) {
		calls += 1
		ensName := "bob.eth"
		return &ensName, nil
	}

	if _, err := service.Resolve(context.Background(), nil, &address, nil); err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		if _, err := service.Resolve(context.Background(), nil, &address, nil); err != nil {
			t.Fatalf("second Resolve returned error: %v", err)
		}
		if calls == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected cache expiry to trigger a second lookup, got %d calls", calls)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestENSResolveBatchUsesCache(t *testing.T) {
	service := newTestENSService(t)
	t.Setenv("RPC_URL_1", "https://env-rpc.example")

	originalLookup := resolveENSNameViaRPC
	t.Cleanup(func() {
		resolveENSNameViaRPC = originalLookup
	})

	calls := 0
	resolveENSNameViaRPC = func(ctx context.Context, rpcURL string, address string) (*string, error) {
		calls += 1
		ensName := "alice.eth"
		return &ensName, nil
	}

	address := "0x0000000000000000000000000000000000000001"
	records, err := service.ResolveBatch(context.Background(), ENSRecordsInput{
		Addresses: []string{address, address},
	})
	if err != nil {
		t.Fatalf("ResolveBatch returned error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected one deduplicated record, got %d", len(records))
	}
	if records[0].Name == nil || *records[0].Name != "alice.eth" {
		t.Fatalf("expected alice.eth, got %#v", records[0])
	}

	records, err = service.ResolveBatch(context.Background(), ENSRecordsInput{
		Addresses: []string{address},
	})
	if err != nil {
		t.Fatalf("second ResolveBatch returned error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected one cached record, got %d", len(records))
	}
	if calls != 1 {
		t.Fatalf("expected one lookup call after cached batch, got %d", calls)
	}
}
