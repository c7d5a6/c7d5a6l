package liquipedia

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
)

const maxAssetBytes = 5 << 20 // 5 MiB

// ValidateAssetURL checks that raw is an https liquipedia.net URL suitable for media fetch.
func ValidateAssetURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("url is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid url")
	}
	if u.Scheme != "https" {
		return "", fmt.Errorf("url must use https")
	}
	host := strings.ToLower(u.Hostname())
	if host != AllowedHost && host != "www."+AllowedHost {
		return "", fmt.Errorf("only liquipedia.net links are supported")
	}
	if strings.Trim(u.EscapedPath(), "/") == "" {
		return "", fmt.Errorf("url must include a path")
	}
	u.Scheme = "https"
	u.Host = AllowedHost
	u.Fragment = ""
	return u.String(), nil
}

// FetchBytes downloads a Liquipedia-hosted asset (e.g. player portrait). Does not use the parse API rate limit.
func (c *Client) FetchBytes(ctx context.Context, assetURL string) ([]byte, string, error) {
	canonical, err := ValidateAssetURL(assetURL)
	if err != nil {
		return nil, "", err
	}
	debuglog.Printf("FetchBytes url=%s", canonical)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, canonical, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", c.userAgent())
	req.Header.Set("Accept", "image/*,*/*;q=0.8")
	// Prefer wiki origin so CDN hotlink rules treat this as a first-party fetch.
	req.Header.Set("Referer", "https://"+AllowedHost+"/")

	res, err := c.httpClient().Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("liquipedia asset request failed: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, maxAssetBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("read liquipedia asset: %w", err)
	}
	if len(body) > maxAssetBytes {
		return nil, "", fmt.Errorf("liquipedia asset exceeds %d bytes", maxAssetBytes)
	}
	if res.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("liquipedia asset http %d", res.StatusCode)
	}

	ctype := strings.TrimSpace(res.Header.Get("Content-Type"))
	if i := strings.Index(ctype, ";"); i >= 0 {
		ctype = strings.TrimSpace(ctype[:i])
	}
	if ctype == "" {
		ctype = "application/octet-stream"
	}
	debuglog.Printf("FetchBytes ok bytes=%d contentType=%s", len(body), ctype)
	return body, ctype, nil
}
