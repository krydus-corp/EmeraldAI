package server_test

import (
	"testing"

	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/server"
)

// Improve tests
func TestNew(t *testing.T) {
	e := server.New(server.Config{})
	if e == nil {
		t.Errorf("Server should not be nil")
	}
}
