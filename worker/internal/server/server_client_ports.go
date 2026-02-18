package server

import "context"

// ServerClient defines the communication port from Worker to Server.
type ServerClient interface {
	// GetProviderConfig fetches tool-specific configuration (e.g., API keys for subfinder).
	GetProviderConfig(ctx context.Context, scanID int, toolName string) (*ProviderConfig, error)

	// EnsureWordlistLocal ensures a wordlist file exists locally, downloading if needed.
	EnsureWordlistLocal(ctx context.Context, wordlistName, basePath string) (string, error)

	// PostBatch sends a batch of data to server (used by BatchSender).
	PostBatch(ctx context.Context, scanID, targetID int, dataType string, items []any) error
}
