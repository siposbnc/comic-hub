package scanner

import "testing"

func TestParseFilename(t *testing.T) {
	cases := []struct {
		path string
		want ParsedName
	}{
		{`X\Saga\Saga 001 (2012).cbz`, ParsedName{Series: "Saga", Number: "1", Year: 2012}},
		{`X\Batman\Batman Annual 02.cbz`, ParsedName{Series: "Batman", Number: "Annual 2"}},
		{`X\Invincible\Invincible v01 001 (2003).cbz`, ParsedName{Series: "Invincible", Number: "1", Volume: 1, Year: 2003}},
		{`X\WW\Wonder Woman 750.cbz`, ParsedName{Series: "Wonder Woman", Number: "750"}},
		{`X\Sandman\The Sandman #1 (1989).cbz`, ParsedName{Series: "The Sandman", Number: "1", Year: 1989}},
		{`X\SM2099\Spider-Man 2099 001.cbz`, ParsedName{Series: "Spider-Man 2099", Number: "1"}},
		{`X\Saga\001.cbz`, ParsedName{Series: "Saga", Number: "1"}}, // number-only -> folder is series
		{`X\Watchmen\Watchmen.cbz`, ParsedName{Series: "Watchmen", Number: ""}},
	}
	for _, c := range cases {
		got := ParseFilename(c.path)
		if got != c.want {
			t.Errorf("ParseFilename(%q) = %+v, want %+v", c.path, got, c.want)
		}
	}
}

func TestSortNumber(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"1", 1},
		{"1.5", 1.5},
		{"750", 750},
		{"1.MU", 1},
		{"", 0},
		{"Annual 2", specialBase + 2},
	}
	for _, c := range cases {
		if got := SortNumber(c.in); got != c.want {
			t.Errorf("SortNumber(%q) = %v, want %v", c.in, got, c.want)
		}
	}
	// Specials must sort after regular issues.
	if SortNumber("Annual 1") <= SortNumber("999") {
		t.Error("expected Annual to sort after issue 999")
	}
}
