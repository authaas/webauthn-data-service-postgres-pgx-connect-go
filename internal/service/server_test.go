//revive:disable:package-comments
package service

import (
	"testing"
)

func TestNew(t *testing.T) {
	t.Run("supplies a logger when given none", func(t *testing.T) {
		server := New(nil, &queriesStub{}, pingerStub{}, "db:5432")

		if server.log == nil {
			t.Error("expected a logger")
		}
	})
}
