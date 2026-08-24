package parse

import (
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/PuerkitoBio/goquery"

	"github.com/c7d5a6/c7d5a6l/internal/liquipedia"
	"github.com/c7d5a6/c7d5a6l/internal/model"
)

type rawResult struct {
	result   model.Result
	domIndex int
	unix     *int64 // nil if unknown
}

// Results extracts matchlist + bracket matches, then assigns Order by time (then DOM).
func Results(doc *goquery.Document) ([]model.Result, error) {
	raw := collectRawResults(doc)
	sort.SliceStable(raw, func(i, j int) bool {
		ti, tj := raw[i].unix, raw[j].unix
		switch {
		case ti != nil && tj != nil && *ti != *tj:
			return *ti < *tj
		case ti != nil && tj == nil:
			return true
		case ti == nil && tj != nil:
			return false
		default:
			return raw[i].domIndex < raw[j].domIndex
		}
	})

	out := make([]model.Result, 0, len(raw))
	for i, r := range raw {
		res := r.result
		if res.Stage != nil {
			res.Phase, res.Round = StagePhaseRound(*res.Stage)
		}
		res.Order = i + 1
		out = append(out, res)
	}
	return out, nil
}

func collectRawResults(doc *goquery.Document) []rawResult {
	var (
		out      []rawResult
		domIndex int
		h3, h4   string
		stageTS  *int64
	)

	doc.Find("h3, h4, .group-table-countdown, .brkts-matchlist-match, .brkts-bracket .brkts-match").Each(func(_ int, sel *goquery.Selection) {
		switch {
		case sel.Is("h3"):
			h3 = cleanHeading(sel)
			h4 = ""
			stageTS = nil
		case sel.Is("h4"):
			h4 = cleanHeading(sel)
			stageTS = nil
		case sel.HasClass("group-table-countdown"):
			stageTS = timerUnix(sel.Find(".timer-object").First())
		case sel.HasClass("brkts-matchlist-match"):
			domIndex++
			if r, ok := parseMatchlistMatch(sel, joinStage(h3, h4, matchlistSubheader(sel)), stageTS, domIndex); ok {
				out = append(out, r)
			}
		case sel.HasClass("brkts-match"):
			if sel.Closest(".brkts-matchlist").Length() > 0 {
				return
			}
			domIndex++
			stage := bracketStage(sel, h3, h4)
			if r, ok := parseBracketMatch(sel, stage, stageTS, domIndex); ok {
				out = append(out, r)
			}
		}
	})

	return out
}

func parseMatchlistMatch(match *goquery.Selection, stage string, stageTS *int64, domIndex int) (rawResult, bool) {
	opponents := match.ChildrenFiltered(".brkts-matchlist-opponent")
	if opponents.Length() < 2 {
		return rawResult{}, false
	}

	scores := match.ChildrenFiltered(".brkts-matchlist-score")
	scoreA := parseScoreCell(scores.Eq(0))
	scoreB := parseScoreCell(scores.Eq(1))

	popup := match.Find(".brkts-match-info-popup").First()
	a := matchSideFromMatchlist(opponents.Eq(0), popup, true)
	b := matchSideFromMatchlist(opponents.Eq(1), popup, false)
	if !matchSideMeaningful(a) && !matchSideMeaningful(b) && scoreA == nil && scoreB == nil {
		// Placeholder / empty shell
		if match.Find(".brkts-matchlist-placeholder-cell").Length() > 0 {
			return rawResult{}, false
		}
	}

	played := scoreA != nil && scoreB != nil
	if !played {
		if fin, ok := popup.Find(".timer-object").First().Attr("data-finished"); ok && fin == "finished" {
			played = scoreA != nil && scoreB != nil
		}
	}

	dt, unix := matchDateTime(popup, stageTS)

	res := model.Result{
		Played:       played,
		ScoreA:       scoreA,
		ScoreB:       scoreB,
		ParticipantA: a,
		ParticipantB: b,
		DateTime:     dt,
	}
	if stage != "" {
		s := stage
		res.Stage = &s
	}

	return rawResult{result: res, domIndex: domIndex, unix: unix}, true
}

func parseBracketMatch(match *goquery.Selection, stage string, stageTS *int64, domIndex int) (rawResult, bool) {
	entries := match.Find(".brkts-opponent-entry")
	if entries.Length() < 2 {
		return rawResult{}, false
	}

	popup := match.Find(".brkts-match-info-popup").First()
	a := matchSideFromBracket(entries.Eq(0), popup, true)
	b := matchSideFromBracket(entries.Eq(1), popup, false)
	if !matchSideMeaningful(a) && !matchSideMeaningful(b) {
		return rawResult{}, false
	}

	scoreA := parseScoreText(cleanText(entries.Eq(0).Find(".brkts-opponent-score-inner").First().Text()))
	scoreB := parseScoreText(cleanText(entries.Eq(1).Find(".brkts-opponent-score-inner").First().Text()))
	played := scoreA != nil && scoreB != nil

	dt, unix := matchDateTime(popup, stageTS)

	res := model.Result{
		Played:       played,
		ScoreA:       scoreA,
		ScoreB:       scoreB,
		ParticipantA: a,
		ParticipantB: b,
		DateTime:     dt,
	}
	if stage != "" {
		s := stage
		res.Stage = &s
	}

	return rawResult{result: res, domIndex: domIndex, unix: unix}, true
}

func matchSideFromMatchlist(opp, popup *goquery.Selection, left bool) *model.Participant {
	name := cleanText(opp.Find(".name").First().Text())
	name = strings.ReplaceAll(name, "\u200b", "")
	name = cleanText(name)

	race := normalizeRace(opp.Find(".race img").First().AttrOr("alt", ""))
	if race == "" {
		race = raceFromClasses(opp)
	}

	var link *string
	if popup.Length() > 0 {
		headerOpp := popup.Find(".match-info-header-opponent").Eq(0)
		if !left {
			headerOpp = popup.Find(".match-info-header-opponent").Eq(1)
		}
		if a := headerOpp.Find(".name a[href]").First(); a.Length() > 0 {
			link = profileURL(a.AttrOr("href", ""))
			if name == "" || strings.EqualFold(name, "TBD") {
				if t := cleanText(a.Text()); t != "" {
					name = t
				}
			}
		}
	}

	if name == "" {
		if label := strings.TrimSpace(opp.AttrOr("aria-label", "")); label != "" {
			name = strings.ReplaceAll(label, "_", " ")
		}
	}
	if name == "" {
		return nil
	}
	if strings.EqualFold(name, "TBD") {
		n := "TBD"
		return &model.Participant{Name: &n}
	}
	if link == nil {
		local := liquipedia.LocalPlayerURL("starcraft", name)
		link = &local
	}

	p := &model.Participant{Name: &name, Link: link}
	if race != "" {
		r := race
		p.Race = &r
	}
	return p
}

func matchSideFromBracket(entry, popup *goquery.Selection, left bool) *model.Participant {
	name := cleanText(entry.Find(".name").First().Text())
	name = strings.ReplaceAll(name, "\u200b", "")
	name = cleanText(name)

	race := normalizeRace(entry.Find(".race img").First().AttrOr("alt", ""))
	if race == "" {
		race = raceFromClasses(entry.Find(".brkts-opponent-entry-left").First())
	}

	var link *string
	if popup.Length() > 0 {
		headerOpp := popup.Find(".match-info-header-opponent").Eq(0)
		if !left {
			headerOpp = popup.Find(".match-info-header-opponent").Eq(1)
		}
		if a := headerOpp.Find(".name a[href]").First(); a.Length() > 0 {
			link = profileURL(a.AttrOr("href", ""))
			if name == "" {
				name = cleanText(a.Text())
			}
		}
	}

	if name == "" {
		if label := strings.TrimSpace(entry.AttrOr("aria-label", "")); label != "" {
			name = strings.ReplaceAll(label, "_", " ")
		}
	}
	if name == "" || strings.EqualFold(name, "TBD") {
		if name == "" {
			return nil
		}
		n := "TBD"
		return &model.Participant{Name: &n}
	}
	if link == nil {
		local := liquipedia.LocalPlayerURL("starcraft", name)
		link = &local
	}

	p := &model.Participant{Name: &name, Link: link}
	if race != "" {
		r := race
		p.Race = &r
	}
	return p
}

func matchSideMeaningful(p *model.Participant) bool {
	if p == nil || p.Name == nil {
		return false
	}
	n := cleanText(*p.Name)
	return n != "" && n != "\u200b"
}

func matchDateTime(popup *goquery.Selection, stageTS *int64) (dt *string, unix *int64) {
	timer := popup.Find(".match-info-countdown .timer-object, .timer-object").First()
	if u := timerUnix(timer); u != nil {
		s := time.Unix(*u, 0).UTC().Format(time.RFC3339)
		return &s, u
	}

	// data-timestamp="error" or missing — try visible text, else stage countdown.
	if text := cleanText(timer.Text()); text != "" {
		if s, u := parseVisibleMatchTime(text); s != nil {
			return s, u
		}
	}

	if stageTS != nil {
		s := time.Unix(*stageTS, 0).UTC().Format(time.RFC3339)
		cp := *stageTS
		return &s, &cp
	}
	return nil, nil
}

func timerUnix(timer *goquery.Selection) *int64 {
	if timer.Length() == 0 {
		return nil
	}
	raw, ok := timer.Attr("data-timestamp")
	if !ok {
		return nil
	}
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "error") {
		return nil
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil
	}
	return &n
}

func parseVisibleMatchTime(text string) (*string, *int64) {
	text = cleanText(text)
	if text == "" {
		return nil, nil
	}

	// "August 18, 2025 - 19:00 KST"
	if i := strings.Index(text, " - "); i > 0 {
		datePart := strings.TrimSpace(text[:i])
		timePart := strings.TrimSpace(text[i+3:])
		timePart = stripTZAbbr(timePart)
		if t, err := time.Parse("January 2, 2006 15:04", datePart+" "+timePart); err == nil {
			// Assume KST (+09) when abbr present; otherwise keep as naive UTC date+time label.
			loc := time.FixedZone("KST", 9*3600)
			if strings.Contains(text, "KST") {
				t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, loc)
			}
			u := t.Unix()
			s := t.UTC().Format(time.RFC3339)
			return &s, &u
		}
	}

	// Date-only: "August 26, 2026"
	if t, err := time.Parse("January 2, 2006", text); err == nil {
		s := t.Format("2006-01-02")
		u := t.Unix()
		return &s, &u
	}
	return nil, nil
}

func stripTZAbbr(s string) string {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return s
	}
	last := fields[len(fields)-1]
	allUpper := true
	for _, r := range last {
		if !unicode.IsLetter(r) || !unicode.IsUpper(r) {
			allUpper = false
			break
		}
	}
	if allUpper && len(last) >= 2 && len(last) <= 4 {
		return strings.Join(fields[:len(fields)-1], " ")
	}
	return s
}

func parseScoreCell(cell *goquery.Selection) *int {
	if cell.Length() == 0 {
		return nil
	}
	content := cell.Find(".brkts-matchlist-cell-content").First()
	if content.Length() > 0 {
		return parseScoreText(cleanText(content.Text()))
	}
	return parseScoreText(cleanText(cell.Text()))
}

func parseScoreText(s string) *int {
	s = cleanText(s)
	if s == "" || strings.EqualFold(s, "vs") {
		return nil
	}
	// Bracket series sometimes use "1-5" in one cell — take leading int only if pure number.
	n, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &n
}

func cleanHeading(sel *goquery.Selection) string {
	clone := sel.Clone()
	clone.Find(".mw-editsection, .mw-headline-number").Remove()
	return cleanText(clone.Text())
}

func cleanBracketHeader(sel *goquery.Selection) string {
	clone := sel.Clone()
	clone.Find("br").Each(func(_ int, br *goquery.Selection) {
		br.ReplaceWithHtml(" ")
	})
	text := cleanText(clone.Text())
	if i := strings.Index(text, "("); i > 0 {
		text = cleanText(text[:i])
	}
	// Liquipedia often concatenates aliases: "Grand FinalGrand FinalFinalGF"
	for _, marker := range []string{"Grand Final", "Third Place", "3rd Place", "Semifinals", "Quarterfinals", "Finals"} {
		if strings.HasPrefix(text, marker) {
			return marker
		}
		if i := strings.Index(text, marker); i >= 0 && i < 3 {
			return marker
		}
	}
	if i := strings.Index(text, "  "); i > 0 {
		text = cleanText(text[:i])
	}
	return text
}

func bracketStage(match *goquery.Selection, h3, h4 string) string {
	bracket := match.Closest(".brkts-bracket")
	headers := bracket.Find(".brkts-round-header .brkts-header")
	depth := 0
	match.Parents().Each(func(_ int, p *goquery.Selection) {
		if p.HasClass("brkts-round-body") {
			depth++
		}
	})

	round := ""
	if depth > 0 && headers.Length() > 0 {
		idx := headers.Length() - depth
		if idx >= 0 && idx < headers.Length() {
			round = cleanBracketHeader(headers.Eq(idx))
		}
	}
	return joinStage(h3, h4, round)
}

func matchlistSubheader(match *goquery.Selection) string {
	prev := match.PrevAll().Filter(".brkts-matchlist-header").First()
	if prev.Length() == 0 {
		return ""
	}
	return cleanText(prev.Text())
}

func joinStage(parts ...string) string {
	var out []string
	seen := map[string]struct{}{}
	for _, p := range parts {
		p = cleanText(p)
		if p == "" {
			continue
		}
		key := strings.ToLower(p)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	return strings.Join(out, " / ")
}
