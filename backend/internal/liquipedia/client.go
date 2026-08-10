package liquipedia

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	defaultUserAgent = "c7d5a6l/0.1 (https://github.com/c7d5a6/c7d5a6l; local-dev parse fixtures)"
	// Liquipedia API ToS: action=parse at most once per 30s.
	defaultParseInterval = 30 * time.Second
)

// Page is a fetched Liquipedia page payload used by parsers and fixtures.
type Page struct {
	Ref        PageRef
	Title      string
	DisplayTitle string
	HTML       string
	FetchedAt  time.Time
}

type Client struct {
	HTTP       *http.Client
	UserAgent  string
	MinInterval time.Duration

	mu         sync.Mutex
	lastParse  time.Time
}

func NewClient() *Client {
	return &Client{
		HTTP: &http.Client{
			Timeout: 60 * time.Second,
		},
		UserAgent:   defaultUserAgent,
		MinInterval: defaultParseInterval,
	}
}

type parseAPIResponse struct {
	Parse *struct {
		Title        string `json:"title"`
		DisplayTitle string `json:"displaytitle"`
		Text         string `json:"text"`
	} `json:"parse"`
	Error *struct {
		Code string `json:"code"`
		Info string `json:"info"`
	} `json:"error"`
}

// FetchPage downloads rendered HTML for a Liquipedia URL via the MediaWiki API.
func (c *Client) FetchPage(ctx context.Context, pageURL string) (*Page, error) {
	ref, err := ParsePageRef(pageURL)
	if err != nil {
		return nil, err
	}
	return c.FetchPageRef(ctx, ref)
}

// FetchPageRef downloads rendered HTML for a wiki page via action=parse.
func (c *Client) FetchPageRef(ctx context.Context, ref PageRef) (*Page, error) {
	if err := c.waitForParseSlot(ctx); err != nil {
		return nil, err
	}

	endpoint := &url.URL{
		Scheme: "https",
		Host:   AllowedHost,
		Path:   "/" + ref.Wiki + "/api.php",
	}
	q := endpoint.Query()
	q.Set("action", "parse")
	q.Set("format", "json")
	q.Set("formatversion", "2")
	q.Set("prop", "text|displaytitle")
	q.Set("redirects", "1")
	q.Set("page", ref.Title)
	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent())
	req.Header.Set("Accept", "application/json")

	res, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("liquipedia request failed: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 16<<20))
	if err != nil {
		return nil, fmt.Errorf("read liquipedia response: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("liquipedia http %d: %s", res.StatusCode, truncate(string(body), 200))
	}

	var parsed parseAPIResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode liquipedia json: %w", err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("liquipedia api error %s: %s", parsed.Error.Code, parsed.Error.Info)
	}
	if parsed.Parse == nil || parsed.Parse.Text == "" {
		return nil, fmt.Errorf("liquipedia api response missing parse text")
	}

	return &Page{
		Ref:          ref,
		Title:        parsed.Parse.Title,
		DisplayTitle: parsed.Parse.DisplayTitle,
		HTML:         parsed.Parse.Text,
		FetchedAt:    time.Now().UTC(),
	}, nil
}

func (c *Client) waitForParseSlot(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	interval := c.MinInterval
	if interval <= 0 {
		interval = defaultParseInterval
	}

	if !c.lastParse.IsZero() {
		wait := interval - time.Since(c.lastParse)
		if wait > 0 {
			timer := time.NewTimer(wait)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
			}
		}
	}

	c.lastParse = time.Now()
	return nil
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

func (c *Client) userAgent() string {
	if c.UserAgent != "" {
		return c.UserAgent
	}
	return defaultUserAgent
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
