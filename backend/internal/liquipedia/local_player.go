package liquipedia

import (
	"fmt"
	"net/url"
	"strings"
)

// LocalPlayerScheme identifies players without a Liquipedia profile page.
const LocalPlayerScheme = "local"

// LocalPlayerURL builds a stable synthetic player identity for nameless-wiki pages.
// Example: local://starcraft/player/Jeco
func LocalPlayerURL(wiki, name string) string {
	wiki = strings.TrimSpace(wiki)
	if wiki == "" {
		wiki = "starcraft"
	}
	name = strings.TrimSpace(name)
	esc := keepMediaWikiParens(url.PathEscape(name))
	return fmt.Sprintf("%s://%s/player/%s", LocalPlayerScheme, wiki, esc)
}

// IsLocalPlayerURL reports whether link is a synthetic local player identity.
func IsLocalPlayerURL(link string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(link)), LocalPlayerScheme+"://")
}

// NormalizePlayerLink canonicalizes a Liquipedia profile URL or a local:// player URL.
func NormalizePlayerLink(link string) (string, error) {
	link = strings.TrimSpace(link)
	if link == "" {
		return "", fmt.Errorf("player link is required")
	}
	if IsLocalPlayerURL(link) {
		return normalizeLocalPlayerURL(link)
	}
	return ValidateURL(link)
}

func normalizeLocalPlayerURL(link string) (string, error) {
	u, err := url.Parse(link)
	if err != nil {
		return "", fmt.Errorf("invalid local player link")
	}
	wiki := strings.TrimSpace(u.Host)
	if wiki == "" {
		return "", fmt.Errorf("local player link missing wiki")
	}
	path := strings.Trim(u.EscapedPath(), "/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[0] != "player" || parts[1] == "" {
		return "", fmt.Errorf("local player link must be local://{wiki}/player/{name}")
	}
	name, err := url.PathUnescape(parts[1])
	if err != nil {
		return "", fmt.Errorf("invalid local player name")
	}
	name = keepMediaWikiParens(strings.TrimSpace(name))
	if name == "" {
		return "", fmt.Errorf("local player name is required")
	}
	return LocalPlayerURL(wiki, name), nil
}
