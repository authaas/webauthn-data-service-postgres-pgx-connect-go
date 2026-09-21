//revive:disable:package-comments
package service

import (
	"context"

	ops "github.com/authaas/webauthn-schema-postgres-bindings-pgx-go"
)

// Queries is what the handlers run: the generated statements, one per RPC.
type Queries interface {
	Register(ctx context.Context, arg ops.RegisterParams) error
	Login(ctx context.Context, arg ops.LoginParams) (int64, error)
	GetCredential(ctx context.Context, id []byte) (ops.Credential, error)
	SetLoginChallenge(ctx context.Context, arg ops.SetLoginChallengeParams) (int64, error)
}
