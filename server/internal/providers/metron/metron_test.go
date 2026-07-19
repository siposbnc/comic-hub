package metron

import (
	"context"
	"errors"
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
	seriesListJSON = `{"count":1,"next":null,"previous":null,"results":[
		{"id":42,"series":"Nova Tide (2021)","year_began":2021,"issue_count":18}
	]}`

	seriesDetailJSON = `{"id":42,"name":"Nova Tide","year_began":2021,
		"publisher":{"name":"Meteor"},"desc":"A deep-space  salvage crew.","image":"http://img/nt.jpg"}`

	// Two pages: the first points `next` at page 2, the second ends it.
	issuesPage1JSON = `{"count":3,"next":"%s/issue/?series_id=42&page=2","previous":null,"results":[
		{"id":1001,"number":"1","issue":"Nova Tide #1","image":"http://img/nt1.jpg"},
		{"id":1002,"number":"2","issue":"Nova Tide #2","image":"http://img/nt2.jpg"}
	]}`
	issuesPage2JSON = `{"count":3,"next":null,"previous":null,"results":[
		{"id":1003,"number":"3","issue":"Nova Tide #3","image":"http://img/nt3.jpg"}
	]}`

	issueDetailJSON = `{"id":1001,"number":"1","title":"","name":["The Driftwake"],
		"cover_date":"2021-05-01","desc":"It   begins.","image":"http://img/nt1.jpg",
		"credits":[{"creator":"R. Okonkwo","role":[{"name":"Writer"}]},
		           {"creator":"L. Demir","role":[{"name":"Penciller"},{"name":"Inker"}]}],
		"characters":[{"id":7,"name":"Capt. Vega Sarin"},{"id":8,"name":"Salla Demir"}],
		"arcs":[{"id":3,"name":"The Driftwake"}]}`
)

func newTestClient(t *testing.T) (*Client, *string) {
	t.Helper()
	var lastPath string
	var base string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastPath = r.URL.Path
		if u, _, ok := r.BasicAuth(); !ok || u == "" {
			t.Errorf("request missing basic auth: %s", r.URL)
		}
		if r.Header.Get("User-Agent") == "" {
			t.Errorf("request missing User-Agent")
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/series/":
			_, _ = w.Write([]byte(seriesListJSON))
		case r.URL.Path == "/series/42/":
			_, _ = w.Write([]byte(seriesDetailJSON))
		case r.URL.Path == "/issue/" && r.URL.Query().Get("page") != "2":
			_, _ = w.Write([]byte(strings.Replace(issuesPage1JSON, "%s", base, 1)))
		case r.URL.Path == "/issue/" && r.URL.Query().Get("page") == "2":
			_, _ = w.Write([]byte(issuesPage2JSON))
		case strings.HasPrefix(r.URL.Path, "/issue/"):
			_, _ = w.Write([]byte(issueDetailJSON))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(ts.Close)
	base = ts.URL

	c := New("user", "pass", WithBaseURL(ts.URL), WithHTTPClient(ts.Client()), WithMinInterval(0))
	return c, &lastPath
}

// A 429 with a Retry-After is waited out and retried; exhausting the attempts surfaces
// providers.ErrRateLimited so the match service can leave the series resumable.
func TestRateLimitRetries(t *testing.T) {
	var calls int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(seriesDetailJSON))
	}))
	t.Cleanup(ts.Close)
	c := New("user", "pass", WithBaseURL(ts.URL), WithHTTPClient(ts.Client()), WithMinInterval(0))

	m, err := c.SeriesMeta(context.Background(), "42")
	if err != nil {
		t.Fatalf("SeriesMeta after one 429: %v", err)
	}
	if calls != 2 || m.Name != "Nova Tide" {
		t.Fatalf("calls = %d, meta = %+v; want a single retry succeeding", calls, m)
	}
}

func TestRateLimitExhaustedIsClassified(t *testing.T) {
	var calls int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(ts.Close)
	c := New("user", "pass", WithBaseURL(ts.URL), WithHTTPClient(ts.Client()), WithMinInterval(0))

	_, err := c.SeriesMeta(context.Background(), "42")
	if !errors.Is(err, providers.ErrRateLimited) {
		t.Fatalf("err = %v, want ErrRateLimited", err)
	}
	if calls != rateLimitAttempts {
		t.Fatalf("calls = %d, want %d attempts", calls, rateLimitAttempts)
	}
}

func TestRetryAfterParsing(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
	}{
		{"30", 30 * time.Second},
		{"", defaultRetryAfter},
		{"garbage", defaultRetryAfter},
		{"0", defaultRetryAfter},
		{"99999", maxRetryAfter},
	}
	for _, c := range cases {
		if got := retryAfter(c.in); got != c.want {
			t.Errorf("retryAfter(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestSearchSeries(t *testing.T) {
	c, _ := newTestClient(t)
	got, err := c.SearchSeries(context.Background(), "Nova Tide (2021)")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d candidates, want 1", len(got))
	}
	s := got[0]
	if s.ProviderID != "42" || s.Name != "Nova Tide (2021)" || s.Year != 2021 || s.IssueCount != 18 {
		t.Fatalf("bad mapping: %+v", s)
	}
}

func TestSeriesMeta(t *testing.T) {
	c, _ := newTestClient(t)
	m, err := c.SeriesMeta(context.Background(), "42")
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "Nova Tide" || m.Year != 2021 || m.Publisher != "Meteor" {
		t.Fatalf("bad series meta: %+v", m)
	}
	if m.Description != "A deep-space salvage crew." { // whitespace collapsed
		t.Fatalf("desc = %q", m.Description)
	}
}

func TestIssuesPaginates(t *testing.T) {
	c, _ := newTestClient(t)
	got, err := c.Issues(context.Background(), "42")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d issues across pages, want 3", len(got))
	}
	if got[0].ProviderID != "1001" || got[0].Number != "1" {
		t.Fatalf("bad issue mapping: %+v", got[0])
	}
	if got[2].Number != "3" {
		t.Fatalf("second page not followed: %+v", got)
	}
}

func TestIssueDetail(t *testing.T) {
	c, lastPath := newTestClient(t)
	meta, err := c.Issue(context.Background(), "1001")
	if err != nil {
		t.Fatal(err)
	}
	if *lastPath != "/issue/1001/" {
		t.Fatalf("issue path = %q, want /issue/1001/", *lastPath)
	}
	if meta.Title != "The Driftwake" || meta.Number != "1" {
		t.Fatalf("bad title/number: %+v", meta)
	}
	if meta.Summary != "It begins." {
		t.Fatalf("summary = %q", meta.Summary)
	}
	want := time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	if meta.ReleaseDate != want {
		t.Fatalf("release date = %d, want %d", meta.ReleaseDate, want)
	}
	if len(meta.People["writer"]) != 1 || meta.People["writer"][0] != "R. Okonkwo" {
		t.Fatalf("writer credit missing: %+v", meta.People)
	}
	// Multi-role credit splits into separate lowercased roles.
	if len(meta.People["penciller"]) != 1 || len(meta.People["inker"]) != 1 {
		t.Fatalf("split roles wrong: %+v", meta.People)
	}
	if len(meta.Characters) != 2 {
		t.Fatalf("characters = %v, want 2", meta.Characters)
	}
	if len(meta.StoryArcs) != 1 || meta.StoryArcs[0].ProviderID != "3" || meta.StoryArcs[0].Name != "The Driftwake" {
		t.Fatalf("story arcs = %+v", meta.StoryArcs)
	}
}
