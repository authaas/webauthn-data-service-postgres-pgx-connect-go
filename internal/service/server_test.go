//revive:disable:package-comments
package service

import (
	"testing"
)

func TestNewServer(_ *testing.T) {
	New(&queriesStub{}, pingerStub{}, "db:5432")
}
