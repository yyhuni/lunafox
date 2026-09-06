// Package principal contains the non-secret identity attached to an MCP
// request after static-key authentication.
package principal

import "context"

// Principal identifies the owner of an MCP key without carrying the bearer or
// its digest. The user ID is an audit identity only; it is not a data scope.
type Principal struct {
	UserID      int
	KeyRecordID int
}

type contextKey struct{}

// With returns a context carrying the authenticated MCP principal.
func With(ctx context.Context, value Principal) context.Context {
	return context.WithValue(ctx, contextKey{}, value)
}

// From returns the authenticated principal, if one was attached.
func From(ctx context.Context) (Principal, bool) {
	value, ok := ctx.Value(contextKey{}).(Principal)
	return value, ok && value.UserID > 0 && value.KeyRecordID > 0
}
