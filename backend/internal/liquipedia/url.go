package liquipedia

import (
	"fmt"
	"net/url"
	"strings"
)

const AllowedHost = "liquipedia.net"

// PageRef identifies a Liquipedia wiki page.
type PageRef struct {
	Wiki  string // e.g. "starcraft"
	Title string // e.g. "ASL/20"
	URL   string // canonical https URL
}

// ValidateURL checks that raw is an https liquipedia.net URL.
func ValidateURL(raw string) (string, error) {
	ref, err := ParsePageRef(raw)
	if err != nil {
		return "", err
	}
	return ref.URL, nil
}

// ParsePageRef validates and splits a Liquipedia page URL into wiki + title.
func ParsePageRef(raw string) (PageRef, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return PageRef{}, fmt.Errorf("url is required")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return PageRef{}, fmt.Errorf("invalid url")
	}

	if u.Scheme != "https" {
		return PageRef{}, fmt.Errorf("url must use https")
	}

	host := strings.ToLower(u.Hostname())
	if host != AllowedHost && host != "www."+AllowedHost {
		return PageRef{}, fmt.Errorf("only liquipedia.net links are supported")
	}

	path := strings.Trim(u.EscapedPath(), "/")
	if path == "" {
		return PageRef{}, fmt.Errorf("url must include a wiki and page path")
	}

	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return PageRef{}, fmt.Errorf("url must include a wiki and page title")
	}

	wiki := parts[0]
	title, err := url.PathUnescape(parts[1])
	if err != nil {
		return PageRef{}, fmt.Errorf("invalid page title in url")
	}

	canonical := &url.URL{
		Scheme: "https",
		Host:   AllowedHost,
		Path:   "/" + wiki + "/" + title,
	}

	return PageRef{
		Wiki:  wiki,
		Title: title,
		URL:   canonical.String(),
	}, nil
}
