//revive:disable:package-comments
package service

import (
	"github.com/authaas/webauthn-data-bindings-connect-go/webauthn/data/dataconnect"
)

// Server serves webauthn.data.Service over the generated statements.
type Server struct {
	dataconnect.UnimplementedServiceHandler

	queries Queries
	db      Pinger
	address string
}

// New returns a Server over queries, reporting db under StorageCheckName at
// address.
func New(queries Queries, db Pinger, address string) *Server {
	return &Server{queries: queries, db: db, address: address}
}
