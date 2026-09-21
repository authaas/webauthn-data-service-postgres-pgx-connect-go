//revive:disable:package-comments
package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"connectrpc.com/connect/v2"
	"github.com/jackc/pgx/v5/pgtype"

	identitydata "buf.build/gen/go/authaas/identity-data/protocolbuffers/go/identity/data"
	"buf.build/gen/go/authaas/identity/protocolbuffers/go/identity"
	"buf.build/gen/go/authaas/token/protocolbuffers/go/token"
	attestationobject "buf.build/gen/go/authaas/webauthn-credential/protocolbuffers/go/webauthn/attestation"
	attestationresponse "buf.build/gen/go/authaas/webauthn-credential/protocolbuffers/go/webauthn/authenticator/attestation"
	"buf.build/gen/go/authaas/webauthn-credential/protocolbuffers/go/webauthn/ceremony"
	clientdata "buf.build/gen/go/authaas/webauthn-credential/protocolbuffers/go/webauthn/client_data"
	credentialid "buf.build/gen/go/authaas/webauthn-credential/protocolbuffers/go/webauthn/credential"
	relyingparty "buf.build/gen/go/authaas/webauthn-credential/protocolbuffers/go/webauthn/relying_party"
	"buf.build/gen/go/authaas/webauthn-data/protocolbuffers/go/webauthn/data"
	"buf.build/gen/go/authaas/webauthn-data/protocolbuffers/go/webauthn/data/credential"
	"buf.build/gen/go/authaas/webauthn-data/protocolbuffers/go/webauthn/data/login"
	principal "github.com/authaas/identity-pgx-go"
	ops "github.com/authaas/webauthn-schema-postgres-bindings-pgx-go"
)

const (
	// principalID is a canonical UUID.
	principalID = "01234567-89ab-4def-8123-456789abcdef"

	// malformed is a string the key type cannot hold.
	malformed = "not-a-uuid"
)

var (
	credentialID  = []byte("credential-id")
	digest        = []byte("0123456789abcdef0123456789abcdef")
	loginChallenge = []byte("challenge-challenge-challenge-32")
	clientDataJSON = []byte(`{"type":"webauthn.create"}`)
	attestationObj = []byte("attestation-object")
	transports     = []string{"internal"}

	errStore = errors.New("store failed")
)

// queriesStub answers each statement with what the test set, and keeps the
// parameters it was given.
type queriesStub struct {
	credential ops.Credential
	rows       int64
	err        error

	registered ops.RegisterParams
	loggedIn   ops.LoginParams
}

func (q *queriesStub) Register(_ context.Context, arg ops.RegisterParams) error {
	q.registered = arg

	return q.err
}

func (q *queriesStub) Login(_ context.Context, arg ops.LoginParams) (int64, error) {
	q.loggedIn = arg

	return q.rows, q.err
}

func (q *queriesStub) GetCredential(context.Context, []byte) (ops.Credential, error) {
	return q.credential, q.err
}

func (q *queriesStub) SetLoginChallenge(context.Context, ops.SetLoginChallengeParams) (int64, error) {
	return q.rows, q.err
}

// pingerStub answers Ping with what the test set.
type pingerStub struct {
	err error
}

func (p pingerStub) Ping(context.Context) error { return p.err }

// newServer builds a Server on the stubs.
func newServer(queries *queriesStub, db pingerStub) *Server {
	return New(slog.New(slog.DiscardHandler), queries, db, "db:5432")
}

// storedKey is principalID in the stored form.
func storedKey(t *testing.T) pgtype.UUID {
	t.Helper()

	id, err := principal.Key(identity.Principal_builder{Id: principalID}.Build())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return id
}

// storedCredential is a row as GetCredential answers it.
func storedCredential(t *testing.T) ops.Credential {
	t.Helper()

	return ops.Credential{
		ID:                        credentialID,
		PrincipalID:               storedKey(t),
		SignCount:                 3,
		UvInitialized:             true,
		Transports:                transports,
		BackupState:               true,
		AttestationObject:         attestationObj,
		AttestationClientDataJson: clientDataJSON,
		RpID:                      pgtype.Text{String: "example.test", Valid: true},
		CurrentLoginChallenge:     loginChallenge,
	}
}

// identityRecord is the identity half of a Register request.
func identityRecord(id string) *identitydata.Record {
	return identitydata.Record_builder{
		Principal:             identity.Principal_builder{Id: id}.Build(),
		Profile:               identity.Profile_builder{Name: "name", DisplayName: "display"}.Build(),
		CreationDate:          1,
		LastAuthenticatedDate: 1,
		GrantHash:             token.GrantHash_builder{Bytes: digest}.Build(),
	}.Build()
}

// credentialRecord is the credential half of a Register request.
func credentialRecord() *credential.Record {
	return credential.Record_builder{
		Credential: credentialid.ID_builder{Bytes: credentialID}.Build(),
		Principal:  identity.Principal_builder{Id: principalID}.Build(),
		Attestation: attestationresponse.Response_builder{
			ClientDataJson:    clientdata.JSON_builder{Bytes: clientDataJSON}.Build(),
			AttestationObject: attestationobject.Object_builder{Bytes: attestationObj}.Build(),
			Transports:        transports,
		}.Build(),
		RpId: relyingparty.ID_builder{Value: "example.test"}.Build(),
		Login: login.State_builder{
			SignCount:     3,
			BackupState:   true,
			UvInitialized: true,
		}.Build(),
	}.Build()
}

func registerRequest(id string) *data.RegisterRequest {
	return data.RegisterRequest_builder{
		IdentityRecord:   identityRecord(id),
		CredentialRecord: credentialRecord(),
	}.Build()
}

func loginRequest() *data.LoginRequest {
	return data.LoginRequest_builder{
		Credential: credentialid.ID_builder{Bytes: credentialID}.Build(),
		LoginState: login.State_builder{
			Challenge:     ceremony.Challenge_builder{Bytes: loginChallenge}.Build(),
			SignCount:     4,
			BackupState:   true,
			UvInitialized: true,
		}.Build(),
		GrantHash:             token.GrantHash_builder{Bytes: digest}.Build(),
		LastAuthenticatedDate: 2,
	}.Build()
}

func getCredentialRequest() *data.GetCredentialRequest {
	return data.GetCredentialRequest_builder{
		Credential: credentialid.ID_builder{Bytes: credentialID}.Build(),
	}.Build()
}

func setLoginChallengeRequest() *data.SetLoginChallengeRequest {
	return data.SetLoginChallengeRequest_builder{
		Credential: credentialid.ID_builder{Bytes: credentialID}.Build(),
		Challenge:  ceremony.Challenge_builder{Bytes: loginChallenge}.Build(),
	}.Build()
}

// assertCode fails the test unless err is a *connect.Error carrying want.
func assertCode(t *testing.T, err error, want connect.Code) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected %v, got no error", want)
	}

	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		t.Fatalf("expected a *connect.Error, got %v", err)
	}

	if got := connect.CodeOf(err); got != want {
		t.Errorf("expected %v, got %v", want, got)
	}
}
