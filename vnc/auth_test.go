package vnc

import "testing"

func TestAuthNone_Impl(t *testing.T) {
	var raw interface{}
	raw = new(AuthNone)
	if _, ok := raw.(Auth); !ok {
		t.Fatal("AuthNone doesn't implement Auth")
	}
}
