package access

import (
	"context"
	"testing"
)

func TestAllowed(t *testing.T) {
	cases := []struct {
		ceiling, rating string
		want            bool
	}{
		{"", "Adults Only 18+", true}, // no ceiling = unrestricted
		{"Teen", "", true},            // unrated content is always allowed
		{"Teen", "Unknown", true},     // unknown rating = unrated
		{"Teen", "Everyone", true},    // below ceiling
		{"Teen", "Teen", true},        // at ceiling
		{"Teen", "Mature 17+", false}, // above ceiling
		{"Teen", "Adults Only 18+", false},
		{"Mature 17+", "Teen", true},
		{"Everyone", "Teen", false},
		{"adults only 18+", "r18+", true}, // case-insensitive
	}
	for _, c := range cases {
		if got := Allowed(c.ceiling, c.rating); got != c.want {
			t.Errorf("Allowed(%q, %q) = %v, want %v", c.ceiling, c.rating, got, c.want)
		}
	}
}

func TestTierOrdering(t *testing.T) {
	if !(Tier("Everyone") < Tier("Teen") && Tier("Teen") < Tier("Mature 17+") &&
		Tier("Mature 17+") < Tier("Adults Only 18+")) {
		t.Fatal("tier ordering is not strictly increasing across rating bands")
	}
	if Tier("") != 0 || Tier("nonsense") != 0 {
		t.Fatal("unrated/unknown should be tier 0")
	}
}

func TestCeilingContext(t *testing.T) {
	ctx := WithCeiling(context.Background(), "Teen")
	if CeilingFrom(ctx) != "Teen" {
		t.Fatal("ceiling not carried on context")
	}
	if CeilingFrom(context.Background()) != "" {
		t.Fatal("unset ceiling should be empty")
	}
}
