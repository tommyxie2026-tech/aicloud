package authorization

import "testing"

func TestBuiltinRBAC(t *testing.T) {
	if BuiltinRBAC() == nil {
		t.Fatal("builtin RBAC is nil")
	}
}
