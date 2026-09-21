//revive:disable:package-comments
package service

import (
	"bytes"
	"testing"

	"connectrpc.com/connect/v2"
)

func TestLogin(t *testing.T) {
	t.Run("records the login from the request", func(t *testing.T) {
		queries := &queriesStub{rows: 1}
		server := newServer(queries, pingerStub{})

		if _, err := server.Login(t.Context(), loginRequest()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := queries.loggedIn

		if !bytes.Equal(got.ID, credentialID) || !bytes.Equal(got.CurrentLoginChallenge, loginChallenge) {
			t.Errorf("key, challenge = %x, %x, want the request's", got.ID, got.CurrentLoginChallenge)
		}

		if got.SignCount != 4 || !got.BackupState || !got.UvInitialized {
			t.Errorf("state = %v, want the request's", got)
		}

		if !bytes.Equal(got.GrantHash, digest) || got.LastAuthenticatedDate != 2 {
			t.Errorf("identity = %v, want the digest and date 2", got)
		}
	})

	t.Run("refuses when the challenge is not current", func(t *testing.T) {
		server := newServer(&queriesStub{rows: 0}, pingerStub{})

		_, err := server.Login(t.Context(), loginRequest())

		assertCode(t, err, connect.CodeFailedPrecondition)
	})

	t.Run("reports a statement that failed", func(t *testing.T) {
		server := newServer(&queriesStub{err: errStore}, pingerStub{})

		_, err := server.Login(t.Context(), loginRequest())

		assertCode(t, err, connect.CodeInternal)
	})
}
