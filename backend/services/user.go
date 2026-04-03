package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"gorm.io/gorm"

	"github.com/ringecosystem/degov-square/database"
	dbmodels "github.com/ringecosystem/degov-square/database/models"
	"github.com/ringecosystem/degov-square/internal/utils"
	"github.com/wealdtech/go-ens/v3"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

var resolveENSNameViaRPC = func(ctx context.Context, rpcURL string, address string) (*string, error) {
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", rpcURL, err)
	}
	defer client.Close()

	ensName, err := ens.ReverseResolve(client, common.HexToAddress(address))
	if err != nil {
		return nil, err
	}

	return &ensName, nil
}

var resolveENSAddressViaRPC = func(ctx context.Context, rpcURL string, name string) (*string, error) {
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", rpcURL, err)
	}
	defer client.Close()

	address, err := ens.Resolve(client, name)
	if err != nil {
		return nil, err
	}
	if address == (common.Address{}) {
		return nil, errors.New("no resolution")
	}

	resolvedAddress := strings.ToLower(address.Hex())
	return &resolvedAddress, nil
}

type UserService struct {
	db *gorm.DB
}

func NewUserService() *UserService {
	return &UserService{
		db: database.GetDB(),
	}
}

func (s *UserService) Modify(input dbmodels.User) (*dbmodels.User, error) {
	address := strings.ToLower(input.Address)
	// check if address already exists
	var existingUser dbmodels.User
	err := s.db.Where("address = ?", address).First(&existingUser).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Address does not exist, create new user
		user := &dbmodels.User{
			ID:      utils.NextIDString(),
			Address: address,
			EnsName: input.EnsName,
			Email:   input.Email,
		}

		if err := s.db.Create(user).Error; err != nil {
			return nil, fmt.Errorf("error creating user: %w", err)
		}

		return user, nil
	}

	if existingUser.Email != nil && input.Email != nil && *existingUser.Email != *input.Email {
		existingUser.Email = input.Email
		if err := s.db.Save(&existingUser).Error; err != nil {
			return nil, fmt.Errorf("error updating user: %w", err)
		}
	}
	return &existingUser, nil
}

func (s *UserService) Inspect(seed string) (*dbmodels.User, error) {
	var user dbmodels.User
	err := s.db.Where("address = ?", seed).Or("id = ?", seed).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("error finding user: %w", err)
	}
	return &user, nil
}

func (s *UserService) GetENSName(address string) (*string, error) {
	address = strings.ToLower(strings.TrimSpace(address))
	if address == "" {
		return nil, nil
	}

	user, err := s.Inspect(address)
	if err == nil && user != nil && user.EnsName != nil && *user.EnsName != "" {
		slog.Info("ENS name found in cache", "address", address, "ensName", *user.EnsName)
		return user.EnsName, nil
	}

	rpcURLs := getENSRPCURLs()
	var lastErr error

	for _, rpcURL := range rpcURLs {
		slog.Debug("Trying to resolve ENS name via RPC", "rpcURL", rpcURL)

		ensName, err := resolveENSNameViaRPC(context.Background(), rpcURL, address)
		if err != nil {
			if isNoENSResolutionError(err) {
				slog.Info("Address has no reverse resolution", "address", address, "rpcURL", rpcURL)
				return nil, nil
			}
			lastErr = fmt.Errorf("failed to resolve via %s: %w", rpcURL, err)
			slog.Warn("Failed to resolve ENS name", "rpcURL", rpcURL, "err", err)
			continue
		}

		slog.Info("Successfully resolved ENS name", "address", address, "ensName", *ensName, "rpcURL", rpcURL)

		if user == nil {
			user = &dbmodels.User{
				Address: address,
			}
		}
		user.EnsName = ensName

		if _, err := s.Modify(*user); err != nil {
			slog.Error("Failed to save resolved ENS name to database", "address", address, "err", err)
		}

		return ensName, nil
	}

	slog.Error("Failed to resolve ENS name after trying all available RPCs", "address", address, "lastError", lastErr)
	return nil, fmt.Errorf("all RPCs failed: %w", lastErr)
}

func (s *UserService) GetENSAddress(name string) (*string, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil, nil
	}

	rpcURLs := getENSRPCURLs()
	var lastErr error

	for _, rpcURL := range rpcURLs {
		slog.Debug("Trying to resolve ENS address via RPC", "rpcURL", rpcURL, "name", trimmedName)

		address, err := resolveENSAddressViaRPC(context.Background(), rpcURL, trimmedName)
		if err != nil {
			if isNoENSResolutionError(err) {
				slog.Info("ENS name has no address resolution", "name", trimmedName, "rpcURL", rpcURL)
				return nil, nil
			}
			lastErr = fmt.Errorf("failed to resolve via %s: %w", rpcURL, err)
			slog.Warn("Failed to resolve ENS address", "rpcURL", rpcURL, "name", trimmedName, "err", err)
			continue
		}

		slog.Info("Successfully resolved ENS address", "name", trimmedName, "address", *address, "rpcURL", rpcURL)
		return address, nil
	}

	slog.Error("Failed to resolve ENS address after trying all available RPCs", "name", trimmedName, "lastError", lastErr)
	return nil, fmt.Errorf("all RPCs failed: %w", lastErr)
}

func getENSRPCURLs() []string {
	rpcURLs := []string{
		"https://ethereum-rpc.publicnode.com",
	}

	envRPCs := os.Getenv("RPC_URL_1")
	if envRPCs == "" {
		return rpcURLs
	}

	userRPCs := make([]string, 0)
	for _, rpcURL := range strings.Split(envRPCs, ",") {
		trimmed := strings.TrimSpace(rpcURL)
		if trimmed != "" {
			userRPCs = append(userRPCs, trimmed)
		}
	}

	return append(userRPCs, rpcURLs...)
}

func isNoENSResolutionError(err error) bool {
	if err == nil {
		return false
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "no resolution") ||
		strings.Contains(message, "unregistered name") ||
		strings.Contains(message, "no address")
}
