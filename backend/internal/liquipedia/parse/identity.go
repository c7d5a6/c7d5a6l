package parse

import (
	"strings"
	"unicode"

	"github.com/c7d5a6/c7d5a6l/internal/liquipedia"
	"github.com/c7d5a6/c7d5a6l/internal/model"
)

// resolveParticipantIdentity normalizes display name and link for parsed entrants.
// Unlinked players (missing/invalid/redlink pages) may use "ID 한글이름" labels;
// the Latin ID becomes Name and the Hangul suffix becomes RealName.
func resolveParticipantIdentity(name string, wiki string, link *string) (string, *string, *string) {
	name = cleanText(name)
	if name == "" {
		return "", nil, link
	}
	if link != nil && !liquipedia.IsLocalPlayerURL(*link) {
		return name, nil, link
	}

	var realName *string
	if id, real, ok := splitLatinIDKoreanRealName(name); ok {
		name = id
		realName = &real
	}

	if link == nil || liquipedia.IsLocalPlayerURL(*link) {
		local := liquipedia.LocalPlayerURL(wiki, name)
		link = &local
	}
	return name, realName, link
}

// splitLatinIDKoreanRealName splits "LastHerO 김홍철" into ID + real name.
func splitLatinIDKoreanRealName(name string) (id, realName string, ok bool) {
	name = cleanText(name)
	idx := firstHangulIndex(name)
	if idx <= 0 {
		return "", "", false
	}
	id = strings.TrimSpace(name[:idx])
	realName = strings.TrimSpace(name[idx:])
	if id == "" || realName == "" || !isLatinPlayerID(id) {
		return "", "", false
	}
	return id, realName, true
}

func firstHangulIndex(s string) int {
	for i, r := range s {
		if isHangul(r) {
			return i
		}
	}
	return -1
}

func isHangul(r rune) bool {
	return unicode.Is(unicode.Hangul, r)
}

func isLatinPlayerID(s string) bool {
	hasLatin := false
	for _, r := range s {
		if isHangul(r) {
			return false
		}
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			hasLatin = true
		}
	}
	return hasLatin
}

// participantFromIdentity builds a match-side participant with local name splitting.
func participantFromIdentity(name string, link *string, race string) *model.Participant {
	if strings.EqualFold(name, "TBD") {
		n := "TBD"
		return &model.Participant{Name: &n}
	}
	name, realName, link := resolveParticipantIdentity(name, "starcraft", link)
	p := &model.Participant{Name: &name, RealName: realName, Link: link}
	if race != "" {
		r := race
		p.Race = &r
	}
	return p
}
