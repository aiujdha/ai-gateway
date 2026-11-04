// Package routing reserves channel selection and retry policy contracts.
package routing

import "context"

type Channel struct {
	ID       string
	Provider string
	Models   []string
	Group    string
	Priority int
	Weight   int
}

// Router is a contract only: no weighting, health checks or failover run here.
type Router interface {
	Select(ctx context.Context, model, group string, excludedIDs []string) (Channel, error)
}

type RetryPolicy interface {
	ShouldRetry(statusCode, attempt int) bool
}
