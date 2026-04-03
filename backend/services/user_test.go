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

func newTestUserService(t *testing.T) *UserService {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	if err := db.Exec(`
		CREATE TABLE dgv_user (
			id TEXT PRIMARY KEY,
			address TEXT NOT NULL UNIQUE,
			email TEXT,
			ens_name TEXT,
			ctime DATETIME NOT NULL,
			utime DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create user table: %v", err)
	}

	return &UserService{db: db}
}

func TestGetENSNameUsesCachedUserValue(t *testing.T) {
	service := newTestUserService(t)
	cachedName := "cached.eth"
	now := time.Now()

	if err := service.db.Exec(`
		INSERT INTO dgv_user (id, address, ens_name, ctime)
		VALUES (?, ?, ?, ?)
	`, "user-1", "0x1234", cachedName, now).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	originalLookup := resolveENSNameViaRPC
	defer func() {
		resolveENSNameViaRPC = originalLookup
	}()

	resolveENSNameViaRPC = func(ctx context.Context, rpcURL string, address string) (*string, error) {
		t.Fatalf("resolveENSNameViaRPC should not be called when cache is populated")
		return nil, nil
	}

	ensName, err := service.GetENSName("0x1234")
	if err != nil {
		t.Fatalf("GetENSName returned error: %v", err)
	}
	if ensName == nil || *ensName != cachedName {
		t.Fatalf("expected cached ens name %q, got %#v", cachedName, ensName)
	}
}

func TestGetENSAddressFallsBackToLaterRPC(t *testing.T) {
	service := newTestUserService(t)
	t.Setenv("RPC_URL_1", "https://rpc-a.example, https://rpc-b.example")

	originalLookup := resolveENSAddressViaRPC
	defer func() {
		resolveENSAddressViaRPC = originalLookup
	}()

	calls := make([]string, 0, 2)
	resolveENSAddressViaRPC = func(ctx context.Context, rpcURL string, name string) (*string, error) {
		calls = append(calls, rpcURL)
		if rpcURL == "https://rpc-a.example" {
			return nil, errors.New("temporary dial failure")
		}
		address := "0xabc0000000000000000000000000000000000001"
		return &address, nil
	}

	address, err := service.GetENSAddress("alice.eth")
	if err != nil {
		t.Fatalf("GetENSAddress returned error: %v", err)
	}
	if address == nil || *address != "0xabc0000000000000000000000000000000000001" {
		t.Fatalf("expected resolved address, got %#v", address)
	}

	expectedCalls := []string{"https://rpc-a.example", "https://rpc-b.example"}
	if !reflect.DeepEqual(calls, expectedCalls) {
		t.Fatalf("expected lookup calls %v, got %v", expectedCalls, calls)
	}
}

func TestGetENSAddressReturnsNilForUnregisteredName(t *testing.T) {
	service := newTestUserService(t)
	t.Setenv("RPC_URL_1", "https://rpc-a.example")

	originalLookup := resolveENSAddressViaRPC
	defer func() {
		resolveENSAddressViaRPC = originalLookup
	}()

	resolveENSAddressViaRPC = func(ctx context.Context, rpcURL string, name string) (*string, error) {
		return nil, errors.New("unregistered name")
	}

	address, err := service.GetENSAddress("missing.eth")
	if err != nil {
		t.Fatalf("GetENSAddress returned error: %v", err)
	}
	if address != nil {
		t.Fatalf("expected nil address for unregistered name, got %#v", address)
	}
}
