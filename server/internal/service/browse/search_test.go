package browse

import "testing"

func TestBuildMatch(t *testing.T) {
	cases := map[string]string{
		"batman":        "batman*",
		"court of owls": "court* of* owls*",
		"  Bat  man  ":  "Bat* man*",
		"x-men":         "x* men*",      // punctuation splits into tokens
		`"drop tABLE"`:  "drop* tABLE*", // FTS operators/quotes are stripped
		"saga 12":       "saga* 12*",
		"":              "",
		"   ":           "",
		"!!!":           "",        // nothing searchable
		"Amélie":        "Amélie*", // unicode letters kept
	}
	for in, want := range cases {
		if got := buildMatch(in); got != want {
			t.Errorf("buildMatch(%q) = %q, want %q", in, got, want)
		}
	}
}
