package scanner

import (
	"encoding/xml"
	"strconv"
	"strings"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// SidecarData is the metadata to serialize into a ComicInfo.xml (the inverse of the parser).
// Credits are keyed by normalized role name (e.g. "writer", "penciller"); the writer maps
// the common roles onto ComicInfo's dedicated elements.
type SidecarData struct {
	Series      string
	Number      string
	Title       string
	Summary     string
	Publisher   string
	AgeRating   string
	Language    string
	Volume      int
	ReleaseDate int64
	ReadingDir  domain.ReadingDirection
	PageCount   int
	Credits     map[string][]string
	Genres      []string
	Characters  []string
}

// comicInfoWriteXML is the serialized shape — a superset of the parsed fields, with the
// credit roles ComicInfo defines as dedicated elements. Empty fields are omitted.
type comicInfoWriteXML struct {
	XMLName     xml.Name `xml:"ComicInfo"`
	XSI         string   `xml:"xmlns:xsi,attr"`
	XSD         string   `xml:"xmlns:xsd,attr"`
	Title       string   `xml:"Title,omitempty"`
	Series      string   `xml:"Series,omitempty"`
	Number      string   `xml:"Number,omitempty"`
	Volume      string   `xml:"Volume,omitempty"`
	Summary     string   `xml:"Summary,omitempty"`
	Year        string   `xml:"Year,omitempty"`
	Month       string   `xml:"Month,omitempty"`
	Day         string   `xml:"Day,omitempty"`
	Writer      string   `xml:"Writer,omitempty"`
	Penciller   string   `xml:"Penciller,omitempty"`
	Inker       string   `xml:"Inker,omitempty"`
	Colorist    string   `xml:"Colorist,omitempty"`
	Letterer    string   `xml:"Letterer,omitempty"`
	CoverArtist string   `xml:"CoverArtist,omitempty"`
	Editor      string   `xml:"Editor,omitempty"`
	Publisher   string   `xml:"Publisher,omitempty"`
	Genre       string   `xml:"Genre,omitempty"`
	Characters  string   `xml:"Characters,omitempty"`
	AgeRating   string   `xml:"AgeRating,omitempty"`
	LanguageISO string   `xml:"LanguageISO,omitempty"`
	PageCount   string   `xml:"PageCount,omitempty"`
	Manga       string   `xml:"Manga,omitempty"`
}

// roleElements maps normalized provider role names onto ComicInfo credit elements. Names for
// roles that share an element (e.g. "penciler"/"penciller") are merged.
var roleElements = map[string]string{
	"writer":       "Writer",
	"script":       "Writer",
	"penciller":    "Penciller",
	"penciler":     "Penciller",
	"artist":       "Penciller",
	"inker":        "Inker",
	"colorist":     "Colorist",
	"colourist":    "Colorist",
	"letterer":     "Letterer",
	"cover":        "CoverArtist",
	"coverartist":  "CoverArtist",
	"cover artist": "CoverArtist",
	"editor":       "Editor",
}

// BuildComicInfoXML serializes metadata into a ComicInfo.xml document (with XML declaration).
func BuildComicInfoXML(d SidecarData) ([]byte, error) {
	out := comicInfoWriteXML{
		XSI:         "http://www.w3.org/2001/XMLSchema-instance",
		XSD:         "http://www.w3.org/2001/XMLSchema",
		Title:       strings.TrimSpace(d.Title),
		Series:      strings.TrimSpace(d.Series),
		Number:      strings.TrimSpace(d.Number),
		Summary:     strings.TrimSpace(d.Summary),
		Publisher:   strings.TrimSpace(d.Publisher),
		AgeRating:   strings.TrimSpace(d.AgeRating),
		LanguageISO: strings.TrimSpace(d.Language),
		Genre:       strings.Join(d.Genres, ", "),
		Characters:  strings.Join(d.Characters, ", "),
	}
	if d.Volume > 0 {
		out.Volume = strconv.Itoa(d.Volume)
	}
	if d.PageCount > 0 {
		out.PageCount = strconv.Itoa(d.PageCount)
	}
	if d.ReleaseDate > 0 {
		t := time.UnixMilli(d.ReleaseDate).UTC()
		out.Year = strconv.Itoa(t.Year())
		out.Month = strconv.Itoa(int(t.Month()))
		out.Day = strconv.Itoa(t.Day())
	}
	if d.ReadingDir == domain.RTL {
		out.Manga = "YesAndRightToLeft"
	}

	// Fold credits onto their ComicInfo elements (comma-joined, de-duped, order preserved).
	byElement := map[string][]string{}
	for role, names := range d.Credits {
		el, ok := roleElements[strings.ToLower(strings.TrimSpace(role))]
		if !ok {
			continue
		}
		byElement[el] = append(byElement[el], names...)
	}
	set := func(el string) string { return strings.Join(dedupe(byElement[el]), ", ") }
	out.Writer = set("Writer")
	out.Penciller = set("Penciller")
	out.Inker = set("Inker")
	out.Colorist = set("Colorist")
	out.Letterer = set("Letterer")
	out.CoverArtist = set("CoverArtist")
	out.Editor = set("Editor")

	body, err := xml.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), append(body, '\n')...), nil
}

func dedupe(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
