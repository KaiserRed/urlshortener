package auth

import "context"

// PermissionProvider resolves user permissions. In production it is the SSO client.
type PermissionProvider interface {
	IsAdmin(ctx context.Context, userID int64) (bool, error)
}
