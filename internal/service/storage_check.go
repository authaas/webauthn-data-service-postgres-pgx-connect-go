//revive:disable:package-comments
package service

import (
	"context"
	"time"

	"github.com/pbrpc/connect-protos/diagnostics"
)

// StorageCheckName is the diagnostics name the database reports under.
const StorageCheckName = "storage"

// Pinger reports whether the database answers.
type Pinger interface {
	Ping(ctx context.Context) error
}

// StorageCheck reports the database as a dependency. The method value
// satisfies diagnostics.Check.
func (s *Server) StorageCheck(ctx context.Context) (*diagnostics.ServiceDependency, error) {
	err := s.db.Ping(ctx)

	state := ""
	if err != nil {
		state = err.Error()
	}

	return &diagnostics.ServiceDependency{
		Address:     s.address,
		State:       state,
		LastChecked: time.Now().Unix(),
		Details:     map[string]string{"name": "postgres"},
	}, err
}
