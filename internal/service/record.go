//revive:disable:package-comments
package service

import (
	"github.com/jackc/pgx/v5/pgtype"

	attestationobject "buf.build/gen/go/authaas/webauthn-credential/protocolbuffers/go/webauthn/attestation"
	attestationresponse "buf.build/gen/go/authaas/webauthn-credential/protocolbuffers/go/webauthn/authenticator/attestation"
	"buf.build/gen/go/authaas/webauthn-credential/protocolbuffers/go/webauthn/ceremony"
	clientdata "buf.build/gen/go/authaas/webauthn-credential/protocolbuffers/go/webauthn/client_data"
	credentialid "buf.build/gen/go/authaas/webauthn-credential/protocolbuffers/go/webauthn/credential"
	relyingparty "buf.build/gen/go/authaas/webauthn-credential/protocolbuffers/go/webauthn/relying_party"
	"buf.build/gen/go/authaas/webauthn-data/protocolbuffers/go/webauthn/data/credential"
	"buf.build/gen/go/authaas/webauthn-data/protocolbuffers/go/webauthn/data/login"
	principal "github.com/authaas/identity-pgx-go"
	ops "github.com/authaas/webauthn-schema-postgres-bindings-pgx-go"
)

// record answers with the contract's record for a stored row.
func record(row ops.Credential) (*credential.Record, error) {
	key, err := principal.FromKey(row.PrincipalID)
	if err != nil {
		return nil, err
	}

	return credential.Record_builder{
		Credential: credentialid.ID_builder{Bytes: row.ID}.Build(),
		Principal:  key,
		Attestation: attestationresponse.Response_builder{
			ClientDataJson:    clientdata.JSON_builder{Bytes: row.AttestationClientDataJson}.Build(),
			AttestationObject: attestationobject.Object_builder{Bytes: row.AttestationObject}.Build(),
			Transports:        row.Transports,
		}.Build(),
		RpId: rpID(row.RpID),
		Login: login.State_builder{
			Challenge:     challenge(row.CurrentLoginChallenge),
			SignCount:     uint32(row.SignCount),
			BackupState:   row.BackupState,
			UvInitialized: row.UvInitialized,
		}.Build(),
	}.Build(), nil
}

// rpID answers with the RP id as the contract carries it: unset when the
// operator did not retain it.
func rpID(text pgtype.Text) *relyingparty.ID {
	if !text.Valid {
		return nil
	}

	return relyingparty.ID_builder{Value: text.String}.Build()
}

// challenge answers with the login challenge as the contract carries it:
// unset when none is outstanding.
func challenge(bytes []byte) *ceremony.Challenge {
	if bytes == nil {
		return nil
	}

	return ceremony.Challenge_builder{Bytes: bytes}.Build()
}
