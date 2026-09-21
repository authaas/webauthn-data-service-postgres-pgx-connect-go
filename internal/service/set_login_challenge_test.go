//revive:disable:package-comments
package service

import (
	"testing"

	"connectrpc.com/connect/v2"
)

func TestSetLoginChallenge(t *testing.T) {
	t.Run("replaces the challenge", func(t *testing.T) {
		server := newServer(&queriesStub{rows: 1}, pingerStub{})

		if _, err := server.SetLoginChallenge(t.Context(), setLoginChallengeRequest()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("refuses an unknown credential", func(t *testing.T) {
		server := newServer(&queriesStub{rows: 0}, pingerStub{})

		_, err := server.SetLoginChallenge(t.Context(), setLoginChallengeRequest())

		assertCode(t, err, connect.CodeNotFound)
	})

	t.Run("reports a statement that failed", func(t *testing.T) {
		server := newServer(&queriesStub{err: errStore}, pingerStub{})

		_, err := server.SetLoginChallenge(t.Context(), setLoginChallengeRequest())

		assertCode(t, err, connect.CodeInternal)
	})
}
