package discovery

import (
	"testing"
	"time"
)

func TestInfoTXT(t *testing.T) {
	txt := Info{Instance: "Shelf", Port: 8080, Version: "1.2.3", AuthRequired: true}.TXT()
	want := []string{"version=1.2.3", "auth=true"}
	if len(txt) != len(want) {
		t.Fatalf("TXT() = %v, want %v", txt, want)
	}
	for i := range want {
		if txt[i] != want[i] {
			t.Errorf("TXT()[%d] = %q, want %q", i, txt[i], want[i])
		}
	}
}

func TestAdvertiseRequiresInstance(t *testing.T) {
	if _, err := Advertise(Info{Port: 8080}); err == nil {
		t.Fatal("Advertise with empty instance: want error, got nil")
	}
}

// Smoke test: registration on the real network stack comes up and withdraws cleanly.
// A browse round-trip is exercised end-to-end with the client instead — multicast
// visibility inside CI runners is too environment-dependent to assert on here.
func TestAdvertiseAndClose(t *testing.T) {
	adv, err := Advertise(Info{Instance: "comichub-test", Port: 65123, Version: "test"})
	if err != nil {
		t.Skipf("multicast unavailable in this environment: %v", err)
	}
	done := make(chan struct{})
	go func() { adv.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not return within 5s")
	}
}
