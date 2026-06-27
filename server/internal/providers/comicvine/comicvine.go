// Package comicvine implements the providers.Provider interface against the Comic Vine
// API (comicvine.gamespot.com/api). The API key is supplied by the server (never the
// client); requests are throttled to respect Comic Vine's rate limits and carry a
// descriptive User-Agent (Comic Vine blocks empty/default agents).
package comicvine

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/providers"
)

const (
	defaultBaseURL = "https://comicvine.gamespot.com/api"
	defaultUA      = "ComicHub/0.1 (+https://github.com/siposbnc/comic-hub)"
	// Comic Vine caps each resource at 200 requests/hour; ~1 req/sec stays well under.
	defaultMinInterval = time.Second
)

// Comic Vine prefixes typed resource ids: volumes are 4050-, issues 4000-. We store the
// bare numeric id in our candidates and re-apply the prefix when fetching detail.
const issueResourcePrefix = "4000-"

// Client is a Comic Vine metadata provider. The zero value is not usable; use New.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
	ua      string

	mu      sync.Mutex
	last    time.Time
	minWait time.Duration
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the API base URL (used in tests).
func WithBaseURL(u string) Option { return func(c *Client) { c.baseURL = strings.TrimRight(u, "/") } }

// WithHTTPClient overrides the HTTP client.
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.http = h } }

// WithMinInterval sets the minimum spacing between requests (0 disables throttling).
func WithMinInterval(d time.Duration) Option { return func(c *Client) { c.minWait = d } }

// New builds a Comic Vine client with the given API key.
func New(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		http:    &http.Client{Timeout: 15 * time.Second},
		ua:      defaultUA,
		minWait: defaultMinInterval,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Name identifies the provider.
func (c *Client) Name() string { return "comicvine" }

// SearchSeries finds candidate volumes (series) for a free-text query.
func (c *Client) SearchSeries(ctx context.Context, query string) ([]providers.SeriesCandidate, error) {
	params := url.Values{}
	// Comic Vine's name filter is a substring match, so the "(year)"/"Vol. N" qualifiers a
	// scanned series name carries would match nothing — strip them before searching.
	params.Set("filter", "name:"+providers.CleanQuery(query))
	params.Set("field_list", "id,name,start_year,count_of_issues,image,publisher")
	params.Set("limit", "20")

	var vols []cvVolume
	if err := c.get(ctx, "/volumes/", params, &vols); err != nil {
		return nil, err
	}
	out := make([]providers.SeriesCandidate, 0, len(vols))
	for _, v := range vols {
		out = append(out, providers.SeriesCandidate{
			ProviderID: strconv.Itoa(v.ID),
			Name:       v.Name,
			Year:       atoiSafe(v.StartYear),
			Publisher:  v.Publisher.Name,
			IssueCount: v.CountOfIssues,
			CoverURL:   v.Image.OriginalURL,
		})
	}
	return out, nil
}

// Issues lists the issues of a matched volume (series), in issue-number order.
func (c *Client) Issues(ctx context.Context, seriesProviderID string) ([]providers.IssueCandidate, error) {
	params := url.Values{}
	params.Set("filter", "volume:"+seriesProviderID)
	params.Set("field_list", "id,issue_number,name,image")
	params.Set("sort", "issue_number:asc")
	params.Set("limit", "100")

	var issues []cvIssue
	if err := c.get(ctx, "/issues/", params, &issues); err != nil {
		return nil, err
	}
	out := make([]providers.IssueCandidate, 0, len(issues))
	for _, i := range issues {
		out = append(out, providers.IssueCandidate{
			ProviderID: strconv.Itoa(i.ID),
			Number:     i.IssueNumber,
			Title:      i.Name,
			CoverURL:   i.Image.OriginalURL,
		})
	}
	return out, nil
}

// Issue fetches full metadata for one issue (credits, characters, release date).
func (c *Client) Issue(ctx context.Context, issueProviderID string) (providers.IssueMeta, error) {
	params := url.Values{}
	params.Set("field_list", "name,issue_number,cover_date,deck,description,person_credits,character_credits")

	var d cvIssueDetail
	if err := c.get(ctx, "/issue/"+issueResourcePrefix+issueProviderID+"/", params, &d); err != nil {
		return providers.IssueMeta{}, err
	}

	meta := providers.IssueMeta{
		Title:       d.Name,
		Number:      d.IssueNumber,
		Summary:     summary(d.Deck, d.Description),
		ReleaseDate: parseCoverDate(d.CoverDate),
		People:      map[string][]string{},
	}
	for _, p := range d.PersonCredits {
		for _, role := range splitRoles(p.Role) {
			meta.People[role] = append(meta.People[role], p.Name)
		}
	}
	for _, ch := range d.CharacterCredits {
		if ch.Name != "" {
			meta.Characters = append(meta.Characters, ch.Name)
		}
	}
	return meta, nil
}

// --- HTTP plumbing -------------------------------------------------------------------

// envelope is Comic Vine's standard JSON wrapper. `results` is an array for list endpoints
// and an object for detail endpoints, so callers unmarshal it into the right shape.
type envelope struct {
	Error      string          `json:"error"`
	StatusCode int             `json:"status_code"`
	Results    json.RawMessage `json:"results"`
}

func (c *Client) get(ctx context.Context, path string, params url.Values, out any) error {
	if err := c.wait(ctx); err != nil {
		return err
	}
	params.Set("api_key", c.apiKey)
	params.Set("format", "json")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path+"?"+params.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", c.ua)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("comicvine: http %d", resp.StatusCode)
	}

	var env envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return fmt.Errorf("comicvine: decode: %w", err)
	}
	if env.Error != "" && env.Error != "OK" {
		return fmt.Errorf("comicvine: %s", env.Error)
	}
	if len(env.Results) == 0 || string(env.Results) == "null" {
		return nil
	}
	if err := json.Unmarshal(env.Results, out); err != nil {
		return fmt.Errorf("comicvine: decode results: %w", err)
	}
	return nil
}

// wait spaces requests by minWait, honoring context cancellation. Holding the lock across
// the sleep serializes outbound requests, which is what the rate limit wants.
func (c *Client) wait(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.minWait <= 0 {
		c.last = time.Now()
		return nil
	}
	if d := c.minWait - time.Since(c.last); d > 0 {
		t := time.NewTimer(d)
		defer t.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
		}
	}
	c.last = time.Now()
	return nil
}

// --- wire types ----------------------------------------------------------------------

type cvImage struct {
	OriginalURL string `json:"original_url"`
}

type cvNamed struct {
	Name string `json:"name"`
}

type cvVolume struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	StartYear     string  `json:"start_year"`
	CountOfIssues int     `json:"count_of_issues"`
	Image         cvImage `json:"image"`
	Publisher     cvNamed `json:"publisher"`
}

type cvIssue struct {
	ID          int     `json:"id"`
	IssueNumber string  `json:"issue_number"`
	Name        string  `json:"name"`
	Image       cvImage `json:"image"`
}

type cvCredit struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

type cvIssueDetail struct {
	Name             string     `json:"name"`
	IssueNumber      string     `json:"issue_number"`
	CoverDate        string     `json:"cover_date"`
	Deck             string     `json:"deck"`
	Description      string     `json:"description"`
	PersonCredits    []cvCredit `json:"person_credits"`
	CharacterCredits []cvNamed  `json:"character_credits"`
}

// --- small helpers -------------------------------------------------------------------

var reHTMLTag = regexp.MustCompile(`<[^>]*>`)

// summary prefers the short plain-text deck; otherwise it strips tags from the HTML
// description.
func summary(deck, description string) string {
	if s := strings.TrimSpace(deck); s != "" {
		return s
	}
	s := reHTMLTag.ReplaceAllString(description, "")
	s = strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">", "&#39;", "'", "&quot;", `"`, "&nbsp;", " ").Replace(s)
	return strings.TrimSpace(reSpaceRuns.ReplaceAllString(s, " "))
}

var reSpaceRuns = regexp.MustCompile(`\s+`)

// splitRoles turns Comic Vine's "writer, penciler" role string into normalized roles.
func splitRoles(role string) []string {
	parts := strings.Split(role, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if r := strings.ToLower(strings.TrimSpace(p)); r != "" {
			out = append(out, r)
		}
	}
	return out
}

// parseCoverDate parses Comic Vine's "YYYY-MM-DD" (or bare "YYYY") into epoch ms; 0 when
// unparseable.
func parseCoverDate(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.UnixMilli()
	}
	if y := atoiSafe(s); y > 0 {
		return time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	}
	return 0
}

func atoiSafe(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}
