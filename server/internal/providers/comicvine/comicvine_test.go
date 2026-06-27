package comicvine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/providers"
)

// Compile-time check that the client satisfies the Provider interface.
var _ providers.Provider = (*Client)(nil)

const (
	volumesJSON = `{"error":"OK","status_code":1,"results":[
		{"id":42,"name":"Wonder Woman","start_year":"2016","count_of_issues":28,
		 "image":{"original_url":"http://img/ww.jpg"},"publisher":{"name":"DC Comics"}}
	]}`

	issuesJSON = `{"error":"OK","status_code":1,"results":[
		{"id":1001,"issue_number":"1","name":"The Lies Part One","image":{"original_url":"http://img/ww1.jpg"}},
		{"id":1002,"issue_number":"2","name":"The Lies Part Two","image":{"original_url":"http://img/ww2.jpg"}}
	]}`

	issueJSON = `{"error":"OK","status_code":1,"results":{
		"name":"The Lies Part One","issue_number":"1","cover_date":"2016-08-01",
		"deck":"A new era begins.","description":"<p>Long <b>html</b> blurb</p>",
		"person_credits":[{"name":"Greg Rucka","role":"writer"},{"name":"Liam Sharp","role":"penciler, inker"}],
		"character_credits":[{"name":"Wonder Woman"},{"name":"Steve Trevor"}]
	}}`
)

func newTestClient(t *testing.T) (*Client, *string) {
	t.Helper()
	var lastPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastPath = r.URL.Path
		if r.URL.Query().Get("api_key") == "" {
			t.Errorf("request missing api_key: %s", r.URL)
		}
		if r.Header.Get("User-Agent") == "" {
			t.Errorf("request missing User-Agent")
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/volumes/":
			_, _ = w.Write([]byte(volumesJSON))
		case r.URL.Path == "/issues/":
			_, _ = w.Write([]byte(issuesJSON))
		case strings.HasPrefix(r.URL.Path, "/issue/"):
			_, _ = w.Write([]byte(issueJSON))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(ts.Close)

	c := New("test-key", WithBaseURL(ts.URL), WithHTTPClient(ts.Client()), WithMinInterval(0))
	return c, &lastPath
}

func TestSearchSeries(t *testing.T) {
	c, _ := newTestClient(t)
	got, err := c.SearchSeries(context.Background(), "wonder woman")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d candidates, want 1", len(got))
	}
	s := got[0]
	if s.ProviderID != "42" || s.Name != "Wonder Woman" || s.Year != 2016 ||
		s.Publisher != "DC Comics" || s.IssueCount != 28 || s.CoverURL == "" {
		t.Fatalf("bad mapping: %+v", s)
	}
}

func TestIssues(t *testing.T) {
	c, _ := newTestClient(t)
	got, err := c.Issues(context.Background(), "42")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d issues, want 2", len(got))
	}
	if got[0].ProviderID != "1001" || got[0].Number != "1" || got[0].Title != "The Lies Part One" {
		t.Fatalf("bad issue mapping: %+v", got[0])
	}
}

func TestIssueDetail(t *testing.T) {
	c, lastPath := newTestClient(t)
	meta, err := c.Issue(context.Background(), "1001")
	if err != nil {
		t.Fatal(err)
	}
	if *lastPath != "/issue/4000-1001/" {
		t.Fatalf("issue path = %q, want /issue/4000-1001/", *lastPath)
	}
	if meta.Title != "The Lies Part One" || meta.Number != "1" {
		t.Fatalf("bad title/number: %+v", meta)
	}
	if meta.Summary != "A new era begins." {
		t.Fatalf("deck should win as summary, got %q", meta.Summary)
	}
	want := time.Date(2016, 8, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	if meta.ReleaseDate != want {
		t.Fatalf("release date = %d, want %d", meta.ReleaseDate, want)
	}
	if len(meta.People["writer"]) != 1 || meta.People["writer"][0] != "Greg Rucka" {
		t.Fatalf("writer credit missing: %+v", meta.People)
	}
	// "penciler, inker" must split into two roles, both crediting Liam Sharp.
	if len(meta.People["penciler"]) != 1 || len(meta.People["inker"]) != 1 {
		t.Fatalf("split roles wrong: %+v", meta.People)
	}
	if len(meta.Characters) != 2 {
		t.Fatalf("characters = %v, want 2", meta.Characters)
	}
}

func TestSummaryStripsHTML(t *testing.T) {
	if got := summary("", "<p>Long <b>html</b>&amp;more</p>"); got != "Long html&more" {
		t.Fatalf("summary = %q", got)
	}
}
