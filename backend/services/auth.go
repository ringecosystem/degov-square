package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/patrickmn/go-cache"
	"github.com/ringecosystem/degov-apps/database"
	dbmodels "github.com/ringecosystem/degov-apps/database/models"
	gqlmodels "github.com/ringecosystem/degov-apps/graph/models"
	"github.com/ringecosystem/degov-apps/internal/config"
	"github.com/ringecosystem/degov-apps/types"
	"github.com/spruceid/siwe-go"
	"gorm.io/gorm"
)

type AuthService struct {
	db          *gorm.DB
	nonceCache  *cache.Cache
	userService *UserService
}

func NewAuthService() *AuthService {
	c := cache.New(3*time.Minute, 5*time.Minute)

	return &AuthService{
		db:          database.GetDB(),
		nonceCache:  c,
		userService: NewUserService(),
	}
}

func (s *AuthService) Nonce(input gqlmodels.GetNonceInput) (string, error) {
	var length int
	if input.Length == nil || *input.Length < 6 {
		length = 32
	} else {
		length = int(*input.Length)
	}
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	nonce := hex.EncodeToString(bytes)

	// put nonce in cache, expires in 3 minutes
	s.nonceCache.Set(nonce, true, 3*time.Minute)
	return nonce, nil
}

func (s *AuthService) Login(input gqlmodels.LoginInput) (gqlmodels.LoginOutput, error) {
	message, err := siwe.ParseMessage(input.Message)
	if err != nil {
		err = fmt.Errorf("parse message err: %v", err)
		return gqlmodels.LoginOutput{}, err
	}
	verify, err := message.ValidNow()
	if err != nil {
		err = fmt.Errorf("message valid failed: %v", err)
		return gqlmodels.LoginOutput{}, err
	}

	if !verify {
		err = fmt.Errorf("verify message fail")
		return gqlmodels.LoginOutput{}, err
	}

	//# must open
	nonce := message.GetNonce()
	slog.Debug("login nonce", "nonce", nonce)
	enableCheckNonce := true
	cfg := config.GetConfig()
	if cfg.GetAppEnv().IsDevelopment() {
		enableCheckNonce = cfg.GetStringWithDefault("UNSAFE_ENABLE_VERIFY_NONCE_ON_LOGIN", "true") == "true"
	}
	if enableCheckNonce {
		// check nonce in cache
		_, found := s.nonceCache.Get(nonce)
		if !found {
			err = fmt.Errorf("invalid or expired nonce")
			return gqlmodels.LoginOutput{}, err
		}
		// After nonce verification, delete it from cache (one-time use)
		s.nonceCache.Delete(nonce)
	}

	user, err := s.userService.Modify(dbmodels.User{
		Address: message.GetAddress().Hex(),
	})
	if err != nil {
		err = fmt.Errorf("modify user failed: %v", err)
		return gqlmodels.LoginOutput{}, err
	}

	useSessInfo := types.UserSessInfo{
		Id:      user.ID,
		Address: user.Address,
		Email:   user.Email,
		CTime:   user.CTime,
		UTime:   user.UTime,
	}

	// Create JWT token with claims
	claims := jwt.MapClaims{
		"user": useSessInfo,
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// read secret key from environment variable
	secretKey := []byte(config.GetStringRequired("JWT_SECRET"))
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		err = fmt.Errorf("generate token failed: %v", err)
		return gqlmodels.LoginOutput{}, err
	}

	return gqlmodels.LoginOutput{
		Token: tokenString,
	}, nil
}
