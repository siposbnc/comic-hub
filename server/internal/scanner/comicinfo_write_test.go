package scanner

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

func TestBuildComicInfoXMLRoundTrip(t *testing.T) {
	release := time.Date(2016, 8, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	xml, err := BuildComicInfoXML(SidecarData{
		Series:      "Wonder Woman",
		Number:      "1",
		Title:       "The Lies",
		Summary:     "A new era.",
		Publisher:   "DC Comics",
		Volume:      5,
		ReleaseDate: release,
		ReadingDir:  domain.RTL,
		PageCount:   22,
		Credits: map[string][]string{
			"writer":    {"Greg Rucka"},
			"penciller": {"Liam Sharp"},
			"inker":     {"Liam Sharp"},
		},
		Genres:     []string{"Superhero", "Action"},
		Characters: []string{"Wonder Woman", "Steve Trevor"},
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	// The parsed subset must come back intact.
	ci, err := ParseComicInfo(bytes.NewReader(xml))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ci.Series != "Wonder Woman" || ci.Number != "1" || ci.Title != "The Lies" || ci.Volume != 5 {
		t.Fatalf("round-trip scalars wrong: %+v", ci)
	}
	if ci.ReleaseDate != release {
		t.Fatalf("release date = %d, want %d", ci.ReleaseDate, release)
	}
	if ci.ReadingDir != domain.RTL {
		t.Fatalf("manga RTL not preserved: %q", ci.ReadingDir)
	}

	// Credits/genres/characters aren't parsed back, but must be present in the document.
	s := string(xml)
	for _, want := range []string{
		"<Writer>Greg Rucka</Writer>",
		"<Penciller>Liam Sharp</Penciller>",
		"<Inker>Liam Sharp</Inker>",
		"<Genre>Superhero, Action</Genre>",
		"<Characters>Wonder Woman, Steve Trevor</Characters>",
		"<PageCount>22</PageCount>",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in:\n%s", want, s)
		}
	}
}
