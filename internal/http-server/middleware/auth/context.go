package auth

import "context"

type ctxKey int

const (
	uidKey ctxKey = iota
	isAdminKey
)

// UIDFromContext returns the authenticated user ID stored in ctx.
func UIDFromContext(ctx context.Context) (int64, bool) {
	uid, ok := ctx.Value(uidKey).(int64)
	return uid, ok
}

// IsAdminFromContext returns the admin flag stored in ctx.
func IsAdminFromContext(ctx context.Context) (bool, bool) {
	v, ok := ctx.Value(isAdminKey).(bool)
	return v, ok
}
