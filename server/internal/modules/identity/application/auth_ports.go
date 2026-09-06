package application

import "github.com/yyhuni/lunafox/server/internal/auth"

type PasswordVerifier interface {
	VerifyPassword(password, hashed string) bool
}

type TokenProvider interface {
	GenerateTokenPair(userID int, username string, tokenVersion int) (*auth.TokenPair, error)
	ValidateToken(token string) (*auth.Claims, error)
	GenerateAccessToken(userID int, username string, tokenVersion int) (string, int64, error)
}
