//revive:disable:package-comments
package service

import (
	"context"
	stderrors "errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	errors "github.com/pbrpc/connect-errors"

	"buf.build/gen/go/authaas/webauthn-data/protocolbuffers/go/webauthn/data"
	principal "github.com/authaas/identity-pgx-go"
	ops "github.com/authaas/webauthn-schema-postgres-bindings-pgx-go"
)

// Register establishes an identity with its credential, as one statement.
// A unique violation on either aborts it, and nothing is established.
func (s *Server) Register(ctx context.Context, req *data.RegisterRequest) (*data.RegisterResponse, error) {
	identity, credential := req.GetIdentityRecord(), req.GetCredentialRecord()

	id, err := principal.Key(identity.GetPrincipal())
	if err != nil {
		return nil, invalidKey(ctx)
	}

	attestation, state := credential.GetAttestation(), credential.GetLogin()

	err = s.queries.Register(ctx, ops.RegisterParams{
		ID:                        id,
		Name:                      identity.GetProfile().GetName(),
		DisplayName:               identity.GetProfile().GetDisplayName(),
		CreationDate:              identity.GetCreationDate(),
		LastAuthenticatedDate:     identity.GetLastAuthenticatedDate(),
		GrantHash:                 identity.GetGrantHash().GetBytes(),
		ID_2:                      credential.GetCredential().GetBytes(),
		SignCount:                 int64(state.GetSignCount()),
		UvInitialized:             state.GetUvInitialized(),
		BackupState:               state.GetBackupState(),
		AttestationObject:         attestation.GetAttestationObject().GetBytes(),
		AttestationClientDataJson: attestation.GetClientDataJson().GetBytes(),
		Transports:                attestation.GetTransports(),
		RpID:                      pgtype.Text{String: credential.GetRpId().GetValue(), Valid: credential.HasRpId()},
	})

	var pgErr *pgconn.PgError
	if stderrors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
		return nil, errors.AlreadyExists(ctx, "already established", pgErr.ConstraintName, "")
	}

	if err != nil {
		return nil, storeFailed(ctx, "register", err)
	}

	return &data.RegisterResponse{}, nil
}
