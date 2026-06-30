package password

import (
	"errors"
	"strings"
	"testing"
)

func TestHashVerifyRoundTrip(t *testing.T) {
	h, err := Hash("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$") {
		t.Fatalf("unexpected hash format: %q", h)
	}
	if err := Verify("correct horse battery staple", h); err != nil {
		t.Fatalf("verify correct: %v", err)
	}
	if err := Verify("wrong password", h); !errors.Is(err, ErrMismatch) {
		t.Fatalf("verify wrong = %v, want ErrMismatch", err)
	}
}

func TestHashIsSaltedUnique(t *testing.T) {
	a, _ := Hash("same")
	b, _ := Hash("same")
	if a == b {
		t.Fatal("two hashes of the same password should differ (random salt)")
	}
}

func TestVerifyRejectsMalformed(t *testing.T) {
	for _, bad := range []string{"", "notahash", "$argon2id$v=19$bad$x$y", "$bcrypt$..."} {
		if err := Verify("x", bad); err == nil {
			t.Fatalf("expected error for malformed hash %q", bad)
		}
	}
}
