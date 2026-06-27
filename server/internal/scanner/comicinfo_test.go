package scanner

import (
	"strings"
	"testing"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

func TestParseComicInfo(t *testing.T) {
	const xml = `<?xml version="1.0"?>
<ComicInfo>
  <Series>Wonder Woman</Series>
  <Number>1</Number>
  <Title>The Lasso</Title>
  <Volume>2</Volume>
  <Year>1987</Year><Month>2</Month><Day>14</Day>
  <Publisher>DC Comics</Publisher>
  <AgeRating>Teen</AgeRating>
  <LanguageISO>en</LanguageISO>
  <Manga>YesAndRightToLeft</Manga>
  <Pages>
    <Page Image="0" Type="FrontCover"/>
    <Page Image="14" Type="Story" DoublePage="true"/>
  </Pages>
</ComicInfo>`

	ci, err := ParseComicInfo(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ci.Series != "Wonder Woman" || ci.Number != "1" || ci.Title != "The Lasso" {
		t.Errorf("basic fields wrong: %+v", ci)
	}
	if ci.Volume != 2 || ci.Publisher != "DC Comics" || ci.AgeRating != "Teen" || ci.Language != "en" {
		t.Errorf("metadata fields wrong: %+v", ci)
	}
	if ci.ReadingDir != domain.RTL {
		t.Errorf("reading dir = %q, want rtl", ci.ReadingDir)
	}
	if ci.ReleaseDate == 0 {
		t.Error("expected a release date")
	}
	if ci.Pages[0].Type != "FrontCover" {
		t.Errorf("page 0 type = %q", ci.Pages[0].Type)
	}
	if !ci.Pages[14].Double || ci.Pages[14].Type != "Story" {
		t.Errorf("page 14 = %+v", ci.Pages[14])
	}
}

func TestParseComicInfoMessy(t *testing.T) {
	// Empty/missing numerics must not error; reading dir defaults to unset.
	const xml = `<ComicInfo><Series>Saga</Series><Number>1.MU</Number><Volume></Volume><Year></Year></ComicInfo>`
	ci, err := ParseComicInfo(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ci.Series != "Saga" || ci.Number != "1.MU" {
		t.Errorf("fields wrong: %+v", ci)
	}
	if ci.Volume != 0 || ci.ReleaseDate != 0 || ci.ReadingDir != "" {
		t.Errorf("expected zero/unset optionals, got %+v", ci)
	}
}
