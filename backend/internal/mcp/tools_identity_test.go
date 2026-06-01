package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ringecosystem/degov-square/services"
)

type fakeENSService struct {
	resolveRecords map[string]*services.ENSRecord
	resolveErr     error
	delay          time.Duration
	resolveCalls   int
	addresses      []string
}

func (s *fakeENSService) Resolve(ctx context.Context, daoCode *string, address *string, name *string) (*services.ENSRecord, error) {
	s.resolveCalls++
	if address != nil {
		s.addresses = append(s.addresses, *address)
	}
	if s.delay > 0 {
		select {
		case <-time.After(s.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if s.resolveErr != nil {
		return nil, s.resolveErr
	}
	if address == nil {
		return nil, nil
	}
	return s.resolveRecords[*address], nil
}

func TestHydrateAddressIdentitiesResolvesNamesBestEffort(t *testing.T) {
	name := "alice.eth"
	service := &fakeENSService{
		resolveRecords: map[string]*services.ENSRecord{
			"0x0000000000000000000000000000000000000001": {
				Address: stringPtr("0x0000000000000000000000000000000000000001"),
				Name:    &name,
			},
		},
	}

	identities := hydrateAddressIdentities(context.Background(), Config{
		ENSService:        service,
		ENSResolveTimeout: time.Second,
	}, "ring-dao", []string{
		"0x0000000000000000000000000000000000000001",
		"0x0000000000000000000000000000000000000002",
	})

	if got, want := identities["0x0000000000000000000000000000000000000001"].ENSName, &name; got == nil || *got != *want {
		t.Fatalf("ensName = %v, want %v", got, want)
	}
	if got := identities["0x0000000000000000000000000000000000000002"].ENSName; got != nil {
		t.Fatalf("missing ENS name = %v, want nil", got)
	}
	if got, want := service.resolveCalls, 2; got != want {
		t.Fatalf("resolveCalls = %d, want %d", got, want)
	}
}

func TestHydrateAddressIdentitiesIgnoresENSFailure(t *testing.T) {
	service := &fakeENSService{resolveErr: errors.New("rpc failed")}

	identities := hydrateAddressIdentities(context.Background(), Config{
		ENSService:        service,
		ENSResolveTimeout: time.Second,
	}, "ring-dao", []string{"0x0000000000000000000000000000000000000001"})

	identity := identities["0x0000000000000000000000000000000000000001"]
	if identity.Address != "0x0000000000000000000000000000000000000001" {
		t.Fatalf("address = %q, want normalized address", identity.Address)
	}
	if identity.ENSName != nil {
		t.Fatalf("ensName = %v, want nil", identity.ENSName)
	}
}

func TestHydrateAddressIdentitiesBoundsExternalCalls(t *testing.T) {
	service := &fakeENSService{}
	addresses := make([]string, 0, maxAddressIdentityBatch+5)
	for i := 0; i < maxAddressIdentityBatch+5; i++ {
		addresses = append(addresses, "0x0000000000000000000000000000000000000001")
	}

	identities := hydrateAddressIdentities(context.Background(), Config{
		ENSService:        service,
		ENSResolveTimeout: time.Second,
	}, "ring-dao", addresses)

	if got, want := len(identities), 1; got != want {
		t.Fatalf("len(identities) = %d, want %d", got, want)
	}
	if got, want := service.resolveCalls, 1; got != want {
		t.Fatalf("resolveCalls = %d, want %d", got, want)
	}
}
