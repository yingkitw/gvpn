package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

type AuthManager struct {
	tokens      map[string]*TokenInfo
	credentials map[string]string
	mu          sync.RWMutex
	tokenExpiry time.Duration
}

type TokenInfo struct {
	Token     string
	Username  string
	CreatedAt time.Time
	ExpiresAt time.Time
}

func New(tokenExpiry time.Duration) *AuthManager {
	return &AuthManager{
		tokens:      make(map[string]*TokenInfo),
		credentials: make(map[string]string),
		tokenExpiry: tokenExpiry,
	}
}

func (am *AuthManager) AddUser(username, password string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, exists := am.credentials[username]; exists {
		return fmt.Errorf("user already exists")
	}

	am.credentials[username] = hashPassword(password)
	return nil
}

func (am *AuthManager) Authenticate(username, password string) bool {
	am.mu.RLock()
	defer am.mu.RUnlock()

	storedHash, exists := am.credentials[username]
	if !exists {
		return false
	}

	return storedHash == hashPassword(password)
}

func (am *AuthManager) GenerateToken(username string) (string, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, exists := am.credentials[username]; !exists {
		return "", fmt.Errorf("user not found")
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %v", err)
	}

	token := base64.URLEncoding.EncodeToString(tokenBytes)
	now := time.Now()

	am.tokens[token] = &TokenInfo{
		Token:     token,
		Username:  username,
		CreatedAt: now,
		ExpiresAt: now.Add(am.tokenExpiry),
	}

	return token, nil
}

func (am *AuthManager) ValidateToken(token string) (string, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	tokenInfo, exists := am.tokens[token]
	if !exists {
		return "", false
	}

	if time.Now().After(tokenInfo.ExpiresAt) {
		return "", false
	}

	return tokenInfo.Username, true
}

func (am *AuthManager) RevokeToken(token string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	delete(am.tokens, token)
}

func (am *AuthManager) CleanupExpiredTokens() {
	am.mu.Lock()
	defer am.mu.Unlock()

	now := time.Now()
	for token, tokenInfo := range am.tokens {
		if now.After(tokenInfo.ExpiresAt) {
			delete(am.tokens, token)
		}
	}
}

func (am *AuthManager) StartCleanupTask(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			am.CleanupExpiredTokens()
		}
	}()
}

func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return fmt.Sprintf("%x", hash)
}