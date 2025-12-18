// auth/jwt/refresh.go
package jwt

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

// TokenPair contains an access token and refresh token.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// RefreshStore defines the interface for refresh token storage.
type RefreshStore interface {
	// Save stores a refresh token with associated data.
	Save(ctx context.Context, token string, data *RefreshData) error

	// Load retrieves refresh token data.
	Load(ctx context.Context, token string) (*RefreshData, error)

	// Delete removes a refresh token.
	Delete(ctx context.Context, token string) error

	// DeleteBySubject removes all refresh tokens for a subject.
	DeleteBySubject(ctx context.Context, subject string) error
}

// RefreshData contains data associated with a refresh token.
type RefreshData struct {
	Subject   string    `json:"sub"`
	IssuedAt  time.Time `json:"iat"`
	ExpiresAt time.Time `json:"exp"`
	Data      any       `json:"data,omitempty"`
}

// TokenService manages access and refresh tokens.
type TokenService struct {
	signer       *Signer
	store        RefreshStore
	accessTTL    time.Duration
	refreshTTL   time.Duration
	issuer       string
	audience     []string
	claimsFunc   func(subject string, data any) any
}

// TokenServiceConfig configures the token service.
type TokenServiceConfig struct {
	// Signer for creating access tokens. Required.
	Signer *Signer

	// Store for refresh tokens. Required.
	Store RefreshStore

	// AccessTTL is the access token lifetime. Default: 15 minutes.
	AccessTTL time.Duration

	// RefreshTTL is the refresh token lifetime. Default: 7 days.
	RefreshTTL time.Duration

	// Issuer is added to access tokens.
	Issuer string

	// Audience is added to access tokens.
	Audience []string

	// ClaimsFunc creates custom claims from subject and data.
	// If nil, creates standard Claims with Subject set.
	ClaimsFunc func(subject string, data any) any
}

// NewTokenService creates a new token service.
func NewTokenService(cfg TokenServiceConfig) *TokenService {
	if cfg.AccessTTL == 0 {
		cfg.AccessTTL = 15 * time.Minute
	}
	if cfg.RefreshTTL == 0 {
		cfg.RefreshTTL = 7 * 24 * time.Hour
	}

	return &TokenService{
		signer:     cfg.Signer,
		store:      cfg.Store,
		accessTTL:  cfg.AccessTTL,
		refreshTTL: cfg.RefreshTTL,
		issuer:     cfg.Issuer,
		audience:   cfg.Audience,
		claimsFunc: cfg.ClaimsFunc,
	}
}

// CreateTokenPair generates a new access/refresh token pair.
func (s *TokenService) CreateTokenPair(ctx context.Context, subject string, data any) (*TokenPair, error) {
	// Create access token
	accessToken, err := s.createAccessToken(subject, data)
	if err != nil {
		return nil, err
	}

	// Create refresh token
	refreshToken, err := generateRefreshToken()
	if err != nil {
		return nil, err
	}

	// Store refresh token
	refreshData := &RefreshData{
		Subject:   subject,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(s.refreshTTL),
		Data:      data,
	}

	if err := s.store.Save(ctx, refreshToken, refreshData); err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

// RefreshTokens validates a refresh token and creates new tokens.
func (s *TokenService) RefreshTokens(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// Load and validate refresh token
	data, err := s.store.Load(ctx, refreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	if time.Now().After(data.ExpiresAt) {
		s.store.Delete(ctx, refreshToken)
		return nil, ErrTokenExpired
	}

	// Delete old refresh token (rotation)
	s.store.Delete(ctx, refreshToken)

	// Create new token pair
	return s.CreateTokenPair(ctx, data.Subject, data.Data)
}

// RevokeRefreshToken invalidates a refresh token.
func (s *TokenService) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	return s.store.Delete(ctx, refreshToken)
}

// RevokeAllTokens invalidates all refresh tokens for a subject.
func (s *TokenService) RevokeAllTokens(ctx context.Context, subject string) error {
	return s.store.DeleteBySubject(ctx, subject)
}

// createAccessToken creates a signed access token.
func (s *TokenService) createAccessToken(subject string, data any) (string, error) {
	var claims any

	if s.claimsFunc != nil {
		claims = s.claimsFunc(subject, data)
	} else {
		claims = &Claims{
			Subject:   subject,
			Issuer:    s.issuer,
			Audience:  s.audience,
			IssuedAt:  NewTime(time.Now()),
			ExpiresAt: NewTime(time.Now().Add(s.accessTTL)),
		}
	}

	return s.signer.Sign(claims)
}

// generateRefreshToken creates a cryptographically secure refresh token.
func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// MemoryRefreshStore implements in-memory refresh token storage.
type MemoryRefreshStore struct {
	mu     sync.RWMutex
	tokens map[string]*RefreshData
}

// NewMemoryRefreshStore creates a new in-memory refresh store.
func NewMemoryRefreshStore() *MemoryRefreshStore {
	return &MemoryRefreshStore{
		tokens: make(map[string]*RefreshData),
	}
}

// Save stores a refresh token.
func (s *MemoryRefreshStore) Save(ctx context.Context, token string, data *RefreshData) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[token] = data
	return nil
}

// Load retrieves refresh token data.
func (s *MemoryRefreshStore) Load(ctx context.Context, token string) (*RefreshData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, ok := s.tokens[token]
	if !ok {
		return nil, errors.New("token not found")
	}
	return data, nil
}

// Delete removes a refresh token.
func (s *MemoryRefreshStore) Delete(ctx context.Context, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, token)
	return nil
}

// DeleteBySubject removes all tokens for a subject.
func (s *MemoryRefreshStore) DeleteBySubject(ctx context.Context, subject string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for token, data := range s.tokens {
		if data.Subject == subject {
			delete(s.tokens, token)
		}
	}
	return nil
}
