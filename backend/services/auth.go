package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/patrickmn/go-cache"
	"github.com/ringecosystem/degov-apps/internal"
	"github.com/ringecosystem/degov-apps/types"
	"github.com/spruceid/siwe-go"
	"gorm.io/gorm"
)

type AuthService struct {
	db         *gorm.DB
	nonceCache *cache.Cache
}

func NewAuthService(db *gorm.DB) *AuthService {
	c := cache.New(3*time.Minute, 5*time.Minute)

	return &AuthService{
		db:         db,
		nonceCache: c,
	}
}

func (s *AuthService) Nonce() (string, error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	nonce := hex.EncodeToString(bytes)

	// put nonce in cache, expires in 3 minutes
	s.nonceCache.Set(nonce, true, 3*time.Minute)
	return nonce, nil
}

func (s *AuthService) Login(input types.LoginInput) (types.LoginOutput, error) {
	message, err := siwe.ParseMessage(input.Message)
	if err != nil {
		err = fmt.Errorf("parse message err: %v", err)
		return types.LoginOutput{}, err
	}
	verify, err := message.ValidNow()
	if err != nil {
		err = fmt.Errorf("message valid failed: %v", err)
		return types.LoginOutput{}, err
	}

	if !verify {
		err = fmt.Errorf("verify message fail")
		return types.LoginOutput{}, err
	}

	nonce := message.GetNonce()

	// check nonce in cache
	_, found := s.nonceCache.Get(nonce)
	if !found {
		err = fmt.Errorf("invalid or expired nonce")
		return types.LoginOutput{}, err
	}

	// After nonce verification, delete it from cache (one-time use)
	s.nonceCache.Delete(nonce)

	// Create JWT token with claims
	claims := jwt.MapClaims{
		"address": message.GetAddress().Hex(),
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// read secret key from environment variable
	secretKey := []byte(internal.GetEnvStringRequired("JWT_SECRET"))
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		err = fmt.Errorf("generate token failed: %v", err)
		return types.LoginOutput{}, err
	}

	return types.LoginOutput{
		Token: tokenString,
	}, nil
}
