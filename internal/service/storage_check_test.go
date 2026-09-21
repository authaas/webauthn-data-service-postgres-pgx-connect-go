//revive:disable:package-comments
package service

import (
	"errors"
	"testing"
)

func TestStorageCheck(t *testing.T) {
	t.Run("reports the database reachable", func(t *testing.T) {
		server := newServer(&queriesStub{}, pingerStub{})

		dependency, err := server.StorageCheck(t.Context())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if dependency.GetAddress() != "db:5432" || dependency.GetState() != "" {
			t.Errorf("dependency = %v, want db:5432 with no state", dependency)
		}
	})

	t.Run("reports the database unreachable", func(t *testing.T) {
		down := errors.New("connection refused")
		server := newServer(&queriesStub{}, pingerStub{err: down})

		dependency, err := server.StorageCheck(t.Context())
		if !errors.Is(err, down) {
			t.Fatalf("error = %v, want %v", err, down)
		}

		if dependency.GetState() != down.Error() {
			t.Errorf("state = %q, want %q", dependency.GetState(), down.Error())
		}
	})
}
