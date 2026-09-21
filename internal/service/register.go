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
	store "github.com/authaas/data-connect-go"
	identity "github.com/authaas/identity-connect-go"
	"github.com/authaas/identity-pgx-go/principal"
	"github.com/authaas/webauthn-data-bindings-connect-go/webauthn/data/dataconnect"
	ops "github.com/authaas/webauthn-schema-postgres-bindings-pgx-go"
)

// Register establishes an identity with its credential, as one statement.
// A unique violation on either aborts it, and nothing is established.
func (s *Server) Register(ctx context.Context, req *data.RegisterRequest) (*data.RegisterResponse, error) {
	record, credential := req.GetIdentityRecord(), req.GetCredentialRecord()

	id, err := principal.Key(record.GetPrincipal())
	if err != nil {
		return nil, identity.InvalidPrincipal(ctx)
	}

	attestation, state := credential.GetAttestation(), credential.GetLogin()

	err = s.queries.Register(ctx, ops.RegisterParams{
		ID:                        id,
		Name:                      record.GetProfile().GetName(),
		DisplayName:               record.GetProfile().GetDisplayName(),
		CreationDate:              record.GetCreationDate(),
		LastAuthenticatedDate:     record.GetLastAuthenticatedDate(),
		GrantHash:                 record.GetGrantHash().GetBytes(),
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
		return nil, store.StoreFailed(ctx, dataconnect.ServiceName, "register", err)
	}

	return &data.RegisterResponse{}, nil
}
