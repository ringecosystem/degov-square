package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"gorm.io/gorm"

	"github.com/ringecosystem/degov-apps/database"
	dbmodels "github.com/ringecosystem/degov-apps/database/models"
	"github.com/ringecosystem/degov-apps/internal/utils"
	"github.com/wealdtech/go-ens/v3"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

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
	user, err := s.Inspect(address)
	if err == nil && user != nil && user.EnsName != nil && *user.EnsName != "" {
		slog.Info("ENS name found in cache", "address", address, "ensName", *user.EnsName)
		return user.EnsName, nil
	}

	rpcURLs := []string{
		"https://ethereum-rpc.publicnode.com",
		"https://eth-mainnet.public.blastapi.io",
		"https://ethereum.therpc.io",
		"https://eth.api.onfinality.io/public",
		"https://eth.nodeconnect.org",
	}

	envRPCs := os.Getenv("RPC_URL_1")
	if envRPCs != "" {
		userRPCs := strings.Split(envRPCs, ",")
		rpcURLs = append(userRPCs, rpcURLs...)
	}

	ethAddr := common.HexToAddress(address)
	var lastErr error

	for _, rpcURL := range rpcURLs {
		if rpcURL == "" {
			continue
		}
		slog.Debug("Trying to resolve ENS name via RPC", "rpcURL", rpcURL)

		client, err := ethclient.DialContext(context.Background(), rpcURL)
		if err != nil {
			lastErr = fmt.Errorf("failed to connect to %s: %w", rpcURL, err)
			slog.Warn("Failed to connect to Ethereum endpoint", "rpcURL", rpcURL, "err", err)
			continue
		}
		defer client.Close()

		ensName, err := ens.ReverseResolve(client, ethAddr)
		if err != nil {
			if err.Error() == "no resolution" {
				slog.Info("Address has no reverse resolution", "address", address, "rpcURL", rpcURL)
				return nil, nil
			}
			lastErr = fmt.Errorf("failed to resolve via %s: %w", rpcURL, err)
			slog.Warn("Failed to resolve ENS name", "rpcURL", rpcURL, "err", err)
			continue
		}

		slog.Info("Successfully resolved ENS name", "address", address, "ensName", ensName, "rpcURL", rpcURL)

		if user == nil {
			user = &dbmodels.User{
				Address: address,
			}
		}
		user.EnsName = &ensName

		if _, err := s.Modify(*user); err != nil {
			slog.Error("Failed to save resolved ENS name to database", "address", address, "err", err)
		}

		return &ensName, nil
	}

	slog.Error("Failed to resolve ENS name after trying all available RPCs", "address", address, "lastError", lastErr)
	return nil, fmt.Errorf("all RPCs failed: %w", lastErr)
}
