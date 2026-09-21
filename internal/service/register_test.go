//revive:disable:package-comments
package service

import (
	"bytes"
	"slices"
	"testing"

	"connectrpc.com/connect/v2"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestRegister(t *testing.T) {
	t.Run("establishes both records from the request", func(t *testing.T) {
		queries := &queriesStub{}
		server := newServer(queries, pingerStub{})

		if _, err := server.Register(t.Context(), registerRequest(principalID)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := queries.registered

		if got.ID != storedKey(t) || got.Name != "name" || got.DisplayName != "display" {
			t.Errorf("identity = %v, want the request's", got)
		}

		if got.CreationDate != 1 || got.LastAuthenticatedDate != 1 || !bytes.Equal(got.GrantHash, digest) {
			t.Errorf("identity = %v, want dates 1, 1 and the digest", got)
		}

		if !bytes.Equal(got.ID_2, credentialID) || got.SignCount != 3 || !got.UvInitialized || !got.BackupState {
			t.Errorf("credential = %v, want the request's", got)
		}

		if !bytes.Equal(got.AttestationObject, attestationObj) || !bytes.Equal(got.AttestationClientDataJson, clientDataJSON) {
			t.Errorf("attestation = %v, want the request's", got)
		}

		if !slices.Equal(got.Transports, transports) || !got.RpID.Valid || got.RpID.String != "example.test" {
			t.Errorf("transports, rp id = %v, %v, want the request's", got.Transports, got.RpID)
		}
	})

	t.Run("leaves the RP id unset when the request has none", func(t *testing.T) {
		queries := &queriesStub{}
		server := newServer(queries, pingerStub{})

		request := registerRequest(principalID)
		request.GetCredentialRecord().ClearRpId()

		if _, err := server.Register(t.Context(), request); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if queries.registered.RpID.Valid {
			t.Error("expected no RP id")
		}
	})

	t.Run("refuses a key the type cannot hold", func(t *testing.T) {
		server := newServer(&queriesStub{}, pingerStub{})

		_, err := server.Register(t.Context(), registerRequest(malformed))

		assertCode(t, err, connect.CodeInvalidArgument)
	})

	t.Run("refuses what is already established", func(t *testing.T) {
		taken := &pgconn.PgError{Code: pgerrcode.UniqueViolation, ConstraintName: "credential_pkey"}
		server := newServer(&queriesStub{err: taken}, pingerStub{})

		_, err := server.Register(t.Context(), registerRequest(principalID))

		assertCode(t, err, connect.CodeAlreadyExists)
	})

	t.Run("reports a statement that failed", func(t *testing.T) {
		server := newServer(&queriesStub{err: errStore}, pingerStub{})

		_, err := server.Register(t.Context(), registerRequest(principalID))

		assertCode(t, err, connect.CodeInternal)
	})
}
