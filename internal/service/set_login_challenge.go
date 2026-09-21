//revive:disable:package-comments
package service

import (
	"context"
	"encoding/base64"

	errors "github.com/pbrpc/connect-errors"

	"buf.build/gen/go/authaas/webauthn-data/protocolbuffers/go/webauthn/data"
	ops "github.com/authaas/webauthn-schema-postgres-bindings-pgx-go"
)

// SetLoginChallenge replaces the credential's login challenge, and answers
// once it is durable.
func (s *Server) SetLoginChallenge(
	ctx context.Context, req *data.SetLoginChallengeRequest,
) (*data.SetLoginChallengeResponse, error) {
	id := req.GetCredential().GetBytes()

	rows, err := s.queries.SetLoginChallenge(ctx, ops.SetLoginChallengeParams{
		ID:                    id,
		CurrentLoginChallenge: req.GetChallenge().GetBytes(),
	})
	if err != nil {
		return nil, storeFailed(ctx, "set the login challenge", err)
	}

	if rows == 0 {
		return nil, errors.NotFound(ctx, "credential", base64.RawURLEncoding.EncodeToString(id))
	}

	return &data.SetLoginChallengeResponse{}, nil
}
