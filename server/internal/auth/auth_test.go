package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTManager_GenerateAndValidate(t *testing.T) {
	manager := NewJWTManager("test-secret-key-32-chars-long!!", 15*time.Minute, 7*24*time.Hour)

	// Generate token pair
	pair, err := manager.GenerateTokenPair(1, "admin", 7)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	if pair.AccessToken == "" {
		t.Error("Access token should not be empty")
	}
	if pair.RefreshToken == "" {
		t.Error("Refresh token should not be empty")
	}
	if pair.ExpiresIn != 900 { // 15 minutes = 900 seconds
		t.Errorf("ExpiresIn should be 900, got %d", pair.ExpiresIn)
	}

	// Validate access token
	claims, err := manager.ValidateToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate access token: %v", err)
	}

	if claims.UserID != 1 {
		t.Errorf("UserID should be 1, got %d", claims.UserID)
	}
	if claims.Username != "admin" {
		t.Errorf("Username should be admin, got %s", claims.Username)
	}
	if claims.TokenVersion != 7 {
		t.Errorf("TokenVersion should be 7, got %d", claims.TokenVersion)
	}

	refreshClaims, err := manager.ValidateToken(pair.RefreshToken)
	if err != nil {
		t.Fatalf("validate refresh token: %v", err)
	}
	if refreshClaims.TokenVersion != 7 {
		t.Errorf("Refresh token version should be 7, got %d", refreshClaims.TokenVersion)
	}
}

func TestJWTManager_InvalidToken(t *testing.T) {
	manager := NewJWTManager("test-secret-key-32-chars-long!!", 15*time.Minute, 7*24*time.Hour)

	// Test invalid token
	_, err := manager.ValidateToken("invalid-token")
	if err != ErrInvalidToken {
		t.Errorf("Expected ErrInvalidToken, got %v", err)
	}
}

func TestJWTManager_ExpiredToken(t *testing.T) {
	// Create manager with very short expiration
	manager := NewJWTManager("test-secret-key-32-chars-long!!", 1*time.Millisecond, 7*24*time.Hour)

	// Generate token
	token, _, err := manager.GenerateAccessToken(1, "admin", 0)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Validate expired token
	_, err = manager.ValidateToken(token)
	if err != ErrExpiredToken {
		t.Errorf("Expected ErrExpiredToken, got %v", err)
	}
}

func TestJWTManager_LegacyTokenDefaultsTokenVersionToZero(t *testing.T) {
	const secret = "test-secret-key-32-chars-long!!"
	manager := NewJWTManager(secret, 15*time.Minute, time.Hour)
	legacy := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId":   1,
		"username": "admin",
		"exp":      time.Now().Add(time.Hour).Unix(),
		"iat":      time.Now().Unix(),
		"nbf":      time.Now().Unix(),
	})
	token, err := legacy.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign legacy token: %v", err)
	}

	claims, err := manager.ValidateToken(token)
	if err != nil {
		t.Fatalf("validate legacy token: %v", err)
	}
	if claims.TokenVersion != 0 {
		t.Fatalf("legacy token version = %d, want 0", claims.TokenVersion)
	}
}

func TestHashPassword(t *testing.T) {
	password := "test-password-123"

	// Hash the password
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Hash should start with bcrypt prefix
	if len(hash) < 60 {
		t.Errorf("Hash should be at least 60 chars, got %d", len(hash))
	}

	// Verify the password against the hash
	if !VerifyPassword(password, hash) {
		t.Error("Password verification should pass for correct password")
	}

	// Verify wrong password fails
	if VerifyPassword("wrong-password", hash) {
		t.Error("Password verification should fail for wrong password")
	}
}

func TestHashPassword_Uniqueness(t *testing.T) {
	password := "same-password"

	hash1, _ := HashPassword(password)
	hash2, _ := HashPassword(password)

	// Each hash should be unique due to random salt
	if hash1 == hash2 {
		t.Error("Hashes should be unique due to random salt")
	}

	// But both should verify correctly
	if !VerifyPassword(password, hash1) {
		t.Error("First hash should verify")
	}
	if !VerifyPassword(password, hash2) {
		t.Error("Second hash should verify")
	}
}
