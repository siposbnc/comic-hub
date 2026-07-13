package scanner

import (
	"encoding/xml"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// ComicInfo is the parsed, cleaned subset of a ComicRack/Anansi ComicInfo.xml sidecar
// that Phase 1 consumes (docs/04-server.md §6.3). People/genres/characters are
// normalized into their own tables in a later phase; here we take the denormalized
// book/series fields plus per-page types.
type ComicInfo struct {
	Series      string
	Number      string
	Title       string
	Summary     string
	Publisher   string
	AgeRating   string
	Language    string
	Format      string // ComicInfo <Format> (e.g. "Annual", "TPB", "Variant"); "" when absent
	Volume      int
	ReleaseDate int64                   // epoch ms; 0 when no year
	ReadingDir  domain.ReadingDirection // "" when unspecified
	Pages       map[int]PageMeta        // keyed by image index
}

// PageMeta is per-page info from <Pages><Page .../></Pages>.
type PageMeta struct {
	Type   string
	Double bool
}

type comicInfoXML struct {
	XMLName     xml.Name       `xml:"ComicInfo"`
	Series      string         `xml:"Series"`
	Number      string         `xml:"Number"`
	Title       string         `xml:"Title"`
	Summary     string         `xml:"Summary"`
	Publisher   string         `xml:"Publisher"`
	AgeRating   string         `xml:"AgeRating"`
	LanguageISO string         `xml:"LanguageISO"`
	Format      string         `xml:"Format"`
	Volume      string         `xml:"Volume"`
	Year        string         `xml:"Year"`
	Month       string         `xml:"Month"`
	Day         string         `xml:"Day"`
	Manga       string         `xml:"Manga"`
	Pages       []comicPageXML `xml:"Pages>Page"`
}

type comicPageXML struct {
	Image      string `xml:"Image,attr"`
	Type       string `xml:"Type,attr"`
	DoublePage string `xml:"DoublePage,attr"`
}

// ParseComicInfo decodes a ComicInfo.xml stream into a cleaned ComicInfo.
func ParseComicInfo(r io.Reader) (ComicInfo, error) {
	var x comicInfoXML
	dec := xml.NewDecoder(r)
	if err := dec.Decode(&x); err != nil {
		return ComicInfo{}, err
	}

	ci := ComicInfo{
		Series:    strings.TrimSpace(x.Series),
		Number:    strings.TrimSpace(x.Number),
		Title:     strings.TrimSpace(x.Title),
		Summary:   strings.TrimSpace(x.Summary),
		Publisher: strings.TrimSpace(x.Publisher),
		AgeRating: strings.TrimSpace(x.AgeRating),
		Language:  strings.TrimSpace(x.LanguageISO),
		Format:    strings.TrimSpace(x.Format),
		Volume:    atoiSafe(x.Volume),
	}

	if y := atoiSafe(x.Year); y > 0 {
		month := atoiSafe(x.Month)
		if month < 1 || month > 12 {
			month = 1
		}
		day := atoiSafe(x.Day)
		if day < 1 || day > 31 {
			day = 1
		}
		ci.ReleaseDate = time.Date(y, time.Month(month), day, 0, 0, 0, 0, time.UTC).UnixMilli()
	}

	// Manga "YesAndRightToLeft" is the only value that flips reading direction.
	if strings.EqualFold(x.Manga, "YesAndRightToLeft") {
		ci.ReadingDir = domain.RTL
	}

	if len(x.Pages) > 0 {
		ci.Pages = make(map[int]PageMeta, len(x.Pages))
		for _, p := range x.Pages {
			idx := atoiSafe(p.Image)
			ci.Pages[idx] = PageMeta{
				Type:   strings.TrimSpace(p.Type),
				Double: strings.EqualFold(p.DoublePage, "true"),
			}
		}
	}
	return ci, nil
}

// atoiSafe parses an int, returning 0 for empty or malformed input (ComicInfo files
// in the wild are frequently messy).
func atoiSafe(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}
