// Package metron implements the providers.Provider interface against the Metron API
// (metron.cloud/api). Metron authenticates with HTTP Basic Auth (account username +
// password, supplied server-side), is rate-limited to ~30 requests/minute, and returns
// DRF-paginated JSON. It's a useful complement to Comic Vine — many series Comic Vine
// lacks are on Metron. See https://metron.cloud/wiki/api/.
package metron

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	defaultBaseURL = "https://metron.cloud/api"
	defaultUA      = "ComicHub/0.1 (+https://github.com/siposbnc/comic-hub)"
	// Metron allows 30 requests/minute; ~2s spacing stays under it.
	defaultMinInterval = 2 * time.Second
	// issuePageCap bounds pagination so a huge series can't loop unbounded.
	issuePageCap = 25
)

// Client is a Metron metadata provider. The zero value is not usable; use New.
type Client struct {
	user    string
	pass    string
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

// New builds a Metron client with the given account credentials.
func New(user, pass string, opts ...Option) *Client {
	c := &Client{
		user:    user,
		pass:    pass,
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
func (c *Client) Name() string { return "metron" }

// SearchSeries finds candidate series for a free-text query.
func (c *Client) SearchSeries(ctx context.Context, query string) ([]providers.SeriesCandidate, error) {
	params := url.Values{}
	params.Set("name", providers.CleanQuery(query))

	env, err := c.getList(ctx, "/series/", params)
	if err != nil {
		return nil, err
	}
	var rows []mSeriesList
	if err := json.Unmarshal(env.Results, &rows); err != nil {
		return nil, fmt.Errorf("metron: decode series: %w", err)
	}
	out := make([]providers.SeriesCandidate, 0, len(rows))
	for _, s := range rows {
		out = append(out, providers.SeriesCandidate{
			ProviderID: strconv.Itoa(s.ID),
			Name:       s.Series,
			Year:       s.YearBegan,
			IssueCount: s.IssueCount,
		})
	}
	return out, nil
}

// SeriesMeta fetches series-level detail: description, publisher, start year.
func (c *Client) SeriesMeta(ctx context.Context, seriesProviderID string) (providers.SeriesMeta, error) {
	var d mSeriesDetail
	if err := c.getDetail(ctx, "/series/"+seriesProviderID+"/", &d); err != nil {
		return providers.SeriesMeta{}, err
	}
	return providers.SeriesMeta{
		Name:        d.Name,
		Year:        d.YearBegan,
		Publisher:   d.Publisher.Name,
		Description: cleanText(d.Desc),
		CoverURL:    d.Image,
	}, nil
}

// Issues lists a series' issues (paginated, following `next`), in issue-number order.
func (c *Client) Issues(ctx context.Context, seriesProviderID string) ([]providers.IssueCandidate, error) {
	params := url.Values{}
	params.Set("series_id", seriesProviderID)
	next := c.baseURL + "/issue/?" + params.Encode()

	var out []providers.IssueCandidate
	for page := 0; next != "" && page < issuePageCap; page++ {
		body, err := c.do(ctx, next)
		if err != nil {
			return nil, err
		}
		var env mListEnvelope
		if err := json.Unmarshal(body, &env); err != nil {
			return nil, fmt.Errorf("metron: decode issues: %w", err)
		}
		var rows []mIssueList
		if err := json.Unmarshal(env.Results, &rows); err != nil {
			return nil, fmt.Errorf("metron: decode issues: %w", err)
		}
		for _, i := range rows {
			out = append(out, providers.IssueCandidate{
				ProviderID: strconv.Itoa(i.ID),
				Number:     i.Number,
				Title:      i.Issue,
				CoverURL:   i.Image,
			})
		}
		if env.Next == nil {
			break
		}
		next = *env.Next
	}
	return out, nil
}

// Issue fetches full metadata for one issue (credits, characters, story arcs, release date).
func (c *Client) Issue(ctx context.Context, issueProviderID string) (providers.IssueMeta, error) {
	var d mIssueDetail
	if err := c.getDetail(ctx, "/issue/"+issueProviderID+"/", &d); err != nil {
		return providers.IssueMeta{}, err
	}

	meta := providers.IssueMeta{
		Title:       issueTitle(d),
		Number:      d.Number,
		Summary:     cleanText(d.Desc),
		ReleaseDate: parseDate(d.CoverDate),
		People:      map[string][]string{},
	}
	for _, cr := range d.Credits {
		if cr.Creator == "" {
			continue
		}
		for _, role := range cr.Role {
			r := strings.ToLower(strings.TrimSpace(role.Name))
			if r != "" {
				meta.People[r] = append(meta.People[r], cr.Creator)
			}
		}
	}
	for _, ch := range d.Characters {
		if ch.Name != "" {
			meta.Characters = append(meta.Characters, ch.Name)
		}
	}
	for _, a := range d.Arcs {
		if a.Name != "" {
			meta.StoryArcs = append(meta.StoryArcs, providers.ArcRef{
				ProviderID: strconv.Itoa(a.ID),
				Name:       a.Name,
			})
		}
	}
	return meta, nil
}

// --- HTTP plumbing -------------------------------------------------------------------

// mListEnvelope is Metron's DRF list wrapper (detail endpoints return the object directly).
type mListEnvelope struct {
	Count   int             `json:"count"`
	Next    *string         `json:"next"`
	Results json.RawMessage `json:"results"`
}

func (c *Client) getList(ctx context.Context, path string, params url.Values) (mListEnvelope, error) {
	body, err := c.do(ctx, c.baseURL+path+"?"+params.Encode())
	if err != nil {
		return mListEnvelope{}, err
	}
	var env mListEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return mListEnvelope{}, fmt.Errorf("metron: decode: %w", err)
	}
	return env, nil
}

func (c *Client) getDetail(ctx context.Context, path string, out any) error {
	body, err := c.do(ctx, c.baseURL+path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("metron: decode: %w", err)
	}
	return nil
}

// do performs a throttled, authenticated GET and returns the response body.
func (c *Client) do(ctx context.Context, fullURL string) ([]byte, error) {
	if err := c.wait(ctx); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.ua)
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(c.user, c.pass)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("metron: unauthorized (check METRON_USERNAME/METRON_PASSWORD)")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metron: http %d", resp.StatusCode)
	}
	const maxBody = 8 << 20 // 8 MiB guard
	return io.ReadAll(io.LimitReader(resp.Body, maxBody))
}

// wait spaces requests by minWait, honoring context cancellation.
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

type mGeneric struct {
	Name string `json:"name"`
}

type mResource struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type mSeriesList struct {
	ID         int    `json:"id"`
	Series     string `json:"series"` // display name, e.g. "Batman (2016)"
	YearBegan  int    `json:"year_began"`
	IssueCount int    `json:"issue_count"`
}

type mSeriesDetail struct {
	ID        int      `json:"id"`
	Name      string   `json:"name"`
	YearBegan int      `json:"year_began"`
	Publisher mGeneric `json:"publisher"`
	Desc      string   `json:"desc"`
	Image     string   `json:"image"`
}

type mIssueList struct {
	ID     int    `json:"id"`
	Number string `json:"number"`
	Issue  string `json:"issue"` // display name
	Image  string `json:"image"`
}

type mCredit struct {
	Creator string     `json:"creator"`
	Role    []mGeneric `json:"role"`
}

type mIssueDetail struct {
	ID         int         `json:"id"`
	Number     string      `json:"number"`
	Title      string      `json:"title"` // collected-edition title
	Name       []string    `json:"name"`  // story titles
	CoverDate  string      `json:"cover_date"`
	Desc       string      `json:"desc"`
	Image      string      `json:"image"`
	Credits    []mCredit   `json:"credits"`
	Characters []mResource `json:"characters"`
	Arcs       []mResource `json:"arcs"`
}

// --- small helpers -------------------------------------------------------------------

// issueTitle prefers the issue's story title(s), falling back to a collected-edition title.
func issueTitle(d mIssueDetail) string {
	var titles []string
	for _, t := range d.Name {
		if s := strings.TrimSpace(t); s != "" {
			titles = append(titles, s)
		}
	}
	if len(titles) > 0 {
		return strings.Join(titles, "; ")
	}
	return strings.TrimSpace(d.Title)
}

var reSpaceRuns = regexp.MustCompile(`\s+`)

// cleanText collapses whitespace runs in a description (Metron desc is plain text).
func cleanText(s string) string {
	return strings.TrimSpace(reSpaceRuns.ReplaceAllString(s, " "))
}

// parseDate parses Metron's "YYYY-MM-DD" cover date into epoch ms (0 when unparseable).
func parseDate(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.UnixMilli()
	}
	if y, err := strconv.Atoi(s); err == nil && y > 0 {
		return time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	}
	return 0
}
