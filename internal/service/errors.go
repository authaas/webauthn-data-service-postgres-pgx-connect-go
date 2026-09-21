//revive:disable:package-comments
package service

import (
	"context"

	"git.sonicoriginal.software/logger"
	errors "github.com/pbrpc/connect-errors"
)

const (
	errDomain          = "webauthn.data"
	errCodeStoreFailed = "STORE_FAILED"
)

// storeFailed reports a statement that did not run to completion.
func storeFailed(ctx context.Context, action string, err error) error {
	logger.FromContext(ctx).ErrorContext(ctx, "Failed to "+action, "error", err)

	return errors.Internal(ctx, "failed to "+action, errCodeStoreFailed, errDomain)
}

// invalidKey reports a principal id the key type could not hold.
func invalidKey(ctx context.Context) error {
	return errors.InvalidArgument(ctx, "validation failed", errors.FieldViolation{
		Field:       "principal.id",
		Description: "is not a UUID",
	})
}
