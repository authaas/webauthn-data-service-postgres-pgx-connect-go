//revive:disable:package-comments
package service

import (
	"context"

	errors "github.com/pbrpc/connect-errors"

	"buf.build/gen/go/authaas/webauthn-data/protocolbuffers/go/webauthn/data"
	ops "github.com/authaas/webauthn-schema-postgres-bindings-pgx-go"
)

// Login consumes the exact challenge and records the authentication on the
// credential and its identity, as one statement. Zero rows means the
// challenge was not current, and nothing was written.
func (s *Server) Login(ctx context.Context, req *data.LoginRequest) (*data.LoginResponse, error) {
	state := req.GetLoginState()

	rows, err := s.queries.Login(ctx, ops.LoginParams{
		ID:                    req.GetCredential().GetBytes(),
		CurrentLoginChallenge: state.GetChallenge().GetBytes(),
		SignCount:             int64(state.GetSignCount()),
		BackupState:           state.GetBackupState(),
		UvInitialized:         state.GetUvInitialized(),
		GrantHash:             req.GetGrantHash().GetBytes(),
		LastAuthenticatedDate: req.GetLastAuthenticatedDate(),
	})
	if err != nil {
		return nil, storeFailed(ctx, "record the login", err)
	}

	if rows == 0 {
		return nil, errors.PreconditionFailed(
			ctx, "login not recorded", "CHALLENGE_NOT_CURRENT", "login_state.challenge",
			"no credential with this outstanding challenge",
		)
	}

	return &data.LoginResponse{}, nil
}
