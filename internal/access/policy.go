// Package access reserves authentication and authorization boundaries.
package access

import "context"

type Principal struct {
	UserID string
	KeyID  string
	Group  string
}

type Policy interface {
	Authenticate(ctx context.Context, apiKey string) (Principal, error)
	Authorize(ctx context.Context, principal Principal, model string) error
}
