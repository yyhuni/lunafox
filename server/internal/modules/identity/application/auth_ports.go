package application

import "github.com/yyhuni/lunafox/server/internal/auth"

type PasswordVerifier interface {
	VerifyPassword(password, hashed string) bool
}

type TokenProvider interface {
	GenerateTokenPair(userID int, username string) (*auth.TokenPair, error)
	ValidateToken(token string) (*auth.Claims, error)
	GenerateAccessToken(userID int, username string) (string, int64, error)
}
