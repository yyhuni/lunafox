package application

import (
	"context"
	"errors"

	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
)

// MCPKeyLifecycleService owns status, one-time generation, rotation, and lookup.
type MCPKeyLifecycleService struct {
	store     MCPKeyStore
	generator MCPKeySecretGenerator
}

// NewMCPKeyLifecycleService creates the key lifecycle service.
func NewMCPKeyLifecycleService(store MCPKeyStore, generator MCPKeySecretGenerator) *MCPKeyLifecycleService {
	if store == nil || generator == nil {
		panic("MCP key store and generator are required")
	}
	return &MCPKeyLifecycleService{store: store, generator: generator}
}

// Status returns non-secret key metadata for the authenticated user.
func (service *MCPKeyLifecycleService) Status(ctx context.Context, userID int) (MCPKeyStatus, error) {
	key, err := service.store.GetByUserID(ctx, userID)
	if errors.Is(err, identitydomain.ErrMCPKeyNotFound) {
		return MCPKeyStatus{}, nil
	}
	if err != nil {
		return MCPKeyStatus{}, err
	}
	createdAt := key.CreatedAt
	updatedAt := key.UpdatedAt
	return MCPKeyStatus{Configured: true, CreatedAt: &createdAt, UpdatedAt: &updatedAt}, nil
}

// Generate atomically replaces the user's key and returns its plaintext once.
func (service *MCPKeyLifecycleService) Generate(ctx context.Context, userID int) (*MCPKeyGeneration, error) {
	secret, digest, err := service.generator.Generate()
	if err != nil {
		return nil, errors.Join(identitydomain.ErrMCPKeyGeneration, err)
	}
	key, err := service.store.Replace(ctx, userID, digest)
	if err != nil {
		return nil, err
	}
	return &MCPKeyGeneration{Secret: secret, Configured: true, CreatedAt: key.CreatedAt, UpdatedAt: key.UpdatedAt}, nil
}

// Authenticate resolves a presented secret to non-secret principal metadata.
func (service *MCPKeyLifecycleService) Authenticate(ctx context.Context, secret string) (*identitydomain.MCPKey, error) {
	if secret == "" {
		return nil, identitydomain.ErrMCPKeyNotFound
	}
	return service.store.GetByDigest(ctx, service.generator.Digest(secret))
}

// MCPKeyStatusTimestampsAreUTC normalizes status projections at the boundary.
func MCPKeyStatusTimestampsAreUTC(status MCPKeyStatus) MCPKeyStatus {
	if status.CreatedAt != nil {
		value := status.CreatedAt.UTC()
		status.CreatedAt = &value
	}
	if status.UpdatedAt != nil {
		value := status.UpdatedAt.UTC()
		status.UpdatedAt = &value
	}
	return status
}
