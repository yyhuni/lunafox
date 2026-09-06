package tools

import (
	"context"
	"sync"
)

// CallState is request-local metadata used by transport audit emission. It
// deliberately contains no input, output, credential, or error object.
type CallState struct {
	mu        sync.Mutex
	Tool      string
	Category  string
	ErrorCode string
}

type callStateKey struct{}

// WithCallState attaches an audit-safe state holder to a tool context.
func WithCallState(ctx context.Context, state *CallState) context.Context {
	return context.WithValue(ctx, callStateKey{}, state)
}

// CallStateFromContext returns the request-local state holder.
func CallStateFromContext(ctx context.Context) *CallState {
	state, _ := ctx.Value(callStateKey{}).(*CallState)
	return state
}

// Mark records bounded outcome metadata for audit. Values are controlled by
// the registry and never contain arbitrary domain error text.
func (state *CallState) Mark(category, errorCode string) {
	if state == nil {
		return
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if category != "" {
		state.Category = category
	}
	if errorCode != "" {
		state.ErrorCode = errorCode
	}
}

// Snapshot returns a race-free copy for the transport after the call ends.
func (state *CallState) Snapshot() (tool, category, errorCode string) {
	if state == nil {
		return "", "", ""
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.Tool, state.Category, state.ErrorCode
}
