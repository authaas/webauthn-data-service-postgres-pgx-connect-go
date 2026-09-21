//revive:disable:package-comments
package service

import (
	"context"
	"encoding/base64"
	stderrors "errors"

	"github.com/jackc/pgx/v5"

	errors "github.com/pbrpc/connect-errors"

	"buf.build/gen/go/authaas/webauthn-data/protocolbuffers/go/webauthn/data"
	store "github.com/authaas/data-connect-go"
	"github.com/authaas/webauthn-data-bindings-connect-go/webauthn/data/dataconnect"
)

// GetCredential answers with the credential's record.
func (s *Server) GetCredential(
	ctx context.Context, req *data.GetCredentialRequest,
) (*data.GetCredentialResponse, error) {
	id := req.GetCredential().GetBytes()

	row, err := s.queries.GetCredential(ctx, id)
	if stderrors.Is(err, pgx.ErrNoRows) {
		return nil, errors.NotFound(ctx, "credential", base64.RawURLEncoding.EncodeToString(id))
	}

	if err != nil {
		return nil, store.StoreFailed(ctx, dataconnect.ServiceName, "read the credential", err)
	}

	record, err := record(row)
	if err != nil {
		return nil, store.StoreFailed(ctx, dataconnect.ServiceName, "read the credential", err)
	}

	return data.GetCredentialResponse_builder{Record: record}.Build(), nil
}
