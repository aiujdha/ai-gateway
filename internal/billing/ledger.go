// Package billing reserves the quota lifecycle. No balance or payment is stored.
package billing

import "context"

type Ledger interface {
	Reserve(ctx context.Context, requestID, userID string, estimatedQuota int64) error
	Settle(ctx context.Context, requestID string, actualQuota int64) error
	Refund(ctx context.Context, requestID string) error
}
