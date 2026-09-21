//revive:disable:package-comments
package service

import (
	"bytes"
	"slices"
	"testing"

	"connectrpc.com/connect/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	ops "github.com/authaas/webauthn-schema-postgres-bindings-pgx-go"
)

func TestGetCredential(t *testing.T) {
	t.Run("answers with the record", func(t *testing.T) {
		server := newServer(&queriesStub{credential: storedCredential(t)}, pingerStub{})

		res, err := server.GetCredential(t.Context(), getCredentialRequest())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		record := res.GetRecord()

		if !bytes.Equal(record.GetCredential().GetBytes(), credentialID) {
			t.Errorf("credential = %x, want %x", record.GetCredential().GetBytes(), credentialID)
		}

		if got := record.GetPrincipal().GetId(); got != principalID {
			t.Errorf("principal = %q, want %q", got, principalID)
		}

		attestation := record.GetAttestation()
		if !bytes.Equal(attestation.GetClientDataJson().GetBytes(), clientDataJSON) {
			t.Errorf("client data = %s, want %s", attestation.GetClientDataJson().GetBytes(), clientDataJSON)
		}

		if !bytes.Equal(attestation.GetAttestationObject().GetBytes(), attestationObj) {
			t.Errorf("attestation object = %s, want %s", attestation.GetAttestationObject().GetBytes(), attestationObj)
		}

		if !slices.Equal(attestation.GetTransports(), transports) {
			t.Errorf("transports = %v, want %v", attestation.GetTransports(), transports)
		}

		if got := record.GetRpId().GetValue(); got != "example.test" {
			t.Errorf("rp id = %q, want example.test", got)
		}

		state := record.GetLogin()
		if !bytes.Equal(state.GetChallenge().GetBytes(), loginChallenge) {
			t.Errorf("challenge = %x, want %x", state.GetChallenge().GetBytes(), loginChallenge)
		}

		if state.GetSignCount() != 3 || !state.GetBackupState() || !state.GetUvInitialized() {
			t.Errorf("state = %v, want the row's", state)
		}
	})

	t.Run("leaves the RP id and challenge unset when the row has none", func(t *testing.T) {
		row := storedCredential(t)
		row.RpID = pgtype.Text{}
		row.CurrentLoginChallenge = nil
		server := newServer(&queriesStub{credential: row}, pingerStub{})

		res, err := server.GetCredential(t.Context(), getCredentialRequest())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.GetRecord().HasRpId() || res.GetRecord().GetLogin().HasChallenge() {
			t.Error("expected no RP id and no challenge")
		}
	})

	t.Run("refuses an unknown credential", func(t *testing.T) {
		server := newServer(&queriesStub{err: pgx.ErrNoRows}, pingerStub{})

		_, err := server.GetCredential(t.Context(), getCredentialRequest())

		assertCode(t, err, connect.CodeNotFound)
	})

	t.Run("reports a statement that failed", func(t *testing.T) {
		server := newServer(&queriesStub{err: errStore}, pingerStub{})

		_, err := server.GetCredential(t.Context(), getCredentialRequest())

		assertCode(t, err, connect.CodeInternal)
	})

	t.Run("reports a row whose principal does not read back", func(t *testing.T) {
		server := newServer(&queriesStub{credential: ops.Credential{}}, pingerStub{})

		_, err := server.GetCredential(t.Context(), getCredentialRequest())

		assertCode(t, err, connect.CodeInternal)
	})
}
