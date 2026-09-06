package application

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"

	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

const adminResetPasswordBytes = 24

// PasswordGenerator creates a one-time password for operator delivery.
type PasswordGenerator interface {
	Generate() (string, error)
}

type cryptoPasswordGenerator struct{}

// NewCryptoPasswordGenerator returns the production cryptographic password source.
func NewCryptoPasswordGenerator() PasswordGenerator {
	return cryptoPasswordGenerator{}
}

func (cryptoPasswordGenerator) Generate() (string, error) {
	bytes := make([]byte, adminResetPasswordBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// AdminPasswordResetResult holds the transient credential for the command layer.
type AdminPasswordResetResult struct {
	Password string
}

// AdminPasswordResetService coordinates the privileged administrator reset flow.
type AdminPasswordResetService struct {
	store     AdminPasswordResetStore
	hasher    PasswordHasher
	generator PasswordGenerator
}

// NewAdminPasswordResetService creates the reset service with its required boundaries.
func NewAdminPasswordResetService(store AdminPasswordResetStore, hasher PasswordHasher, generator PasswordGenerator) *AdminPasswordResetService {
	if store == nil {
		panic("admin password reset store is required")
	}
	if hasher == nil {
		panic("admin password reset hasher is required")
	}
	if generator == nil {
		panic("admin password reset password generator is required")
	}
	return &AdminPasswordResetService{store: store, hasher: hasher, generator: generator}
}

// Reset generates and persists a replacement for the existing admin account.
// Plaintext is returned only for the command layer to print after commit.
func (service *AdminPasswordResetService) Reset(ctx context.Context) (*AdminPasswordResetResult, error) {
	if ctx == nil {
		return nil, errors.New("admin password reset context is required")
	}

	password, err := service.generator.Generate()
	if err != nil {
		return nil, err
	}
	if password == "" {
		return nil, errors.New("admin password generator returned an empty password")
	}
	hashedPassword, err := service.hasher.HashPassword(password)
	if err != nil {
		return nil, err
	}
	if err := service.store.ResetAdminPassword(ctx, hashedPassword); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrAdminNotFound
		}
		return nil, err
	}
	return &AdminPasswordResetResult{Password: password}, nil
}
