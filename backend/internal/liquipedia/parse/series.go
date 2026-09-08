package parse

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var (
	reBestOfTitle = regexp.MustCompile(`(?i)best\s+of\s+(\d+)`)
	reBoToken     = regexp.MustCompile(`(?i)\(bo(\d+)\)|\bbo(\d+)\b`)
)

// seriesWinsNeeded is maps one side must win: Bo1→1, Bo3→2, Bo5→3, Bo7→4.
func seriesWinsNeeded(bestOf int) int {
	if bestOf <= 1 {
		return 1
	}
	return bestOf/2 + 1
}

// seriesPlayed reports whether the series is over given map scores and best-of.
// bestOf 0 means unknown format: only treat as complete when a side already has
// two or more maps (Bo3+), not a live 1:0.
func seriesPlayed(scoreA, scoreB *int, bestOf int) bool {
	if scoreA == nil || scoreB == nil {
		return false
	}
	if bestOf <= 0 {
		return *scoreA >= 2 || *scoreB >= 2
	}
	need := seriesWinsNeeded(bestOf)
	return *scoreA >= need || *scoreB >= need
}

func matchSeriesPlayed(match, popup *goquery.Selection, scoreA, scoreB *int) bool {
	bestOf := matchBestOf(match, popup)
	if seriesPlayed(scoreA, scoreB, bestOf) {
		return true
	}
	if bestOf > 0 || scoreA == nil || scoreB == nil {
		return false
	}
	fin, ok := popup.Find(".timer-object").First().Attr("data-finished")
	return ok && fin == "finished"
}

func parseBestOfN(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if m := reBestOfTitle.FindStringSubmatch(s); len(m) == 2 {
		return atoiBestOf(m[1])
	}
	if m := reBoToken.FindStringSubmatch(s); len(m) >= 2 {
		for _, g := range m[1:] {
			if g != "" {
				return atoiBestOf(g)
			}
		}
	}
	return 0
}

func atoiBestOf(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > 19 {
		return 0
	}
	return n
}

func bestOfFromSelection(sel *goquery.Selection) int {
	if sel == nil || sel.Length() == 0 {
		return 0
	}
	var n int
	sel.Find("abbr").EachWithBreak(func(_ int, abbr *goquery.Selection) bool {
		n = parseBestOfN(abbr.AttrOr("title", ""))
		if n == 0 {
			n = parseBestOfN(abbr.Text())
		}
		return n == 0
	})
	if n > 0 {
		return n
	}
	return parseBestOfN(sel.Text())
}

func matchBestOf(match, popup *goquery.Selection) int {
	if n := bestOfFromSelection(popup.Find(".match-info-header-scoreholder-lower")); n > 0 {
		return n
	}
	if n := bestOfFromSelection(popup); n > 0 {
		return n
	}
	if n := bestOfFromSelection(match); n > 0 {
		return n
	}
	if h := bracketRoundHeader(match); h != nil && h.Length() > 0 {
		if n := bestOfFromSelection(h); n > 0 {
			return n
		}
	}
	return 0
}

func bracketRoundHeader(match *goquery.Selection) *goquery.Selection {
	bracket := match.Closest(".brkts-bracket")
	if bracket.Length() == 0 {
		return nil
	}
	headers := bracket.Find(".brkts-round-header .brkts-header")
	depth := 0
	match.Parents().Each(func(_ int, p *goquery.Selection) {
		if p.HasClass("brkts-round-body") {
			depth++
		}
	})
	if depth <= 0 || headers.Length() == 0 {
		return nil
	}
	idx := headers.Length() - depth
	if idx < 0 || idx >= headers.Length() {
		return nil
	}
	return headers.Eq(idx)
}
