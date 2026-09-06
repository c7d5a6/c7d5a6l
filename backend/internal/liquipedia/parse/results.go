package parse

import (
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/PuerkitoBio/goquery"

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
		h2, h3, h4 string
		stageTS  *int64
	)

	doc.Find("h2, h3, h4, .group-table-countdown, .brkts-matchlist-match, .brkts-match-info-flat, .brkts-bracket .brkts-match").Each(func(_ int, sel *goquery.Selection) {
		switch {
		case sel.Is("h2"):
			h2 = cleanHeading(sel)
			h3 = ""
			h4 = ""
			stageTS = nil
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
			stage := resultStageContext(sel, sectionStage(h2, h3, h4, matchlistSubheader(sel)))
			if games := parseMapGamesFromPopup(sel.Find(".brkts-match-info-popup").First(), stage, stageTS, domIndex); len(games) > 0 {
				out = append(out, games...)
				return
			}
			if r, ok := parseMatchlistMatch(sel, stage, stageTS, domIndex); ok {
				out = append(out, r)
			}
		case sel.HasClass("brkts-match-info-flat"):
			if sel.Closest(".brkts-matchlist-match").Length() > 0 {
				return
			}
			domIndex++
			stage := resultStageContext(sel, sectionStage(h2, h3, h4))
			if games := parseMapGamesFromPopup(sel, stage, stageTS, domIndex); len(games) > 0 {
				out = append(out, games...)
			}
		case sel.HasClass("brkts-match"):
			if sel.Closest(".brkts-matchlist").Length() > 0 {
				return
			}
			domIndex++
			stage := resultStageContext(sel, bracketStage(sel, h2, h3, h4))
			if r, ok := parseBracketMatch(sel, stage, stageTS, domIndex); ok {
				out = append(out, r)
			}
		}
	})

	return out
}

func resultStageContext(sel *goquery.Selection, stage string) string {
	if tab := tabGroupLabel(sel); tab != "" {
		return joinStage(stage, tab)
	}
	return stage
}

func tabGroupLabel(sel *goquery.Selection) string {
	content := sel.Closest("[class*='content']")
	if content.Length() == 0 {
		return ""
	}
	class, _ := content.Attr("class")
	tabClass := ""
	for _, part := range strings.Fields(class) {
		if strings.HasPrefix(part, "content") && part != "content" {
			tabClass = "tab" + strings.TrimPrefix(part, "content")
			break
		}
	}
	if tabClass == "" {
		if n := strings.TrimSpace(content.AttrOr("data-count", "")); n != "" {
			tabClass = "tab" + n
		}
	}
	if tabClass == "" {
		return ""
	}
	tabs := content.Closest(".tabs-dynamic")
	if tabs.Length() == 0 {
		return ""
	}
	label := cleanText(tabs.Find("li." + tabClass + " span").First().Text())
	if strings.EqualFold(label, "show all") {
		return ""
	}
	return label
}

// sectionStage prefers h3/h4 headings when present; h2 is used for pages
// like team proleagues where matches sit directly under a Results section.
func sectionStage(h2, h3, h4 string, extra ...string) string {
	if h3 != "" || h4 != "" {
		return joinStage(h3, h4, joinParts(extra))
	}
	return joinStage(h2, h3, h4, joinParts(extra))
}

func joinParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	return joinStage(parts...)
}

func parseMapGamesFromPopup(popup *goquery.Selection, stage string, stageTS *int64, domIndex int) []rawResult {
	if popup.Length() == 0 {
		return nil
	}
	games := popup.Find(".brkts-popup-body-element.brkts-popup-body-game")
	if games.Length() == 0 {
		return nil
	}

	teamStage := joinStage(stage, teamMatchLabel(popup))
	dt, unix := matchDateTime(popup, stageTS)

	var out []rawResult
	games.Each(func(i int, game *goquery.Selection) {
		left := game.Find(".brkts-popup-header-opponent-left .block-player").First()
		right := game.Find(".brkts-popup-header-opponent-right .block-player").First()
		if left.Length() == 0 || right.Length() == 0 {
			return
		}
		a, okA := participantFromBlock(left, "")
		b, okB := participantFromBlock(right, "")
		if !okA || !okB {
			return
		}
		scoreA := parseScoreText(cleanText(game.Find(".brkts-popup-header-opponent-score-left").First().Text()))
		scoreB := parseScoreText(cleanText(game.Find(".brkts-popup-header-opponent-score-right").First().Text()))
		played := scoreA != nil && scoreB != nil

		mapName := cleanText(game.Find(".brkts-popup-spaced a").First().Text())
		matchStage := teamStage
		if mapName != "" {
			matchStage = joinStage(teamStage, mapName)
		}

		res := model.Result{
			Played:       played,
			ScoreA:       scoreA,
			ScoreB:       scoreB,
			ParticipantA: &a,
			ParticipantB: &b,
			DateTime:     dt,
		}
		if matchStage != "" {
			s := matchStage
			res.Stage = &s
		}
		out = append(out, rawResult{result: res, domIndex: domIndex*1000 + i, unix: unix})
	})
	return out
}

func teamMatchLabel(popup *goquery.Selection) string {
	var names []string
	popup.Find(".match-info-header-opponent").Each(func(_ int, opp *goquery.Selection) {
		if name := opponentDisplayName(opp); name != "" {
			names = append(names, name)
		}
	})
	if len(names) >= 2 {
		return names[0] + " vs " + names[1]
	}
	if len(names) == 1 {
		return names[0]
	}
	return ""
}

func opponentDisplayName(opp *goquery.Selection) string {
	if dyn := opp.Find(".team-name-dynamic").First(); dyn.Length() > 0 {
		for _, attr := range []string{"data-team-bracketname", "data-team-shortname", "data-team-name"} {
			if v := cleanText(dyn.AttrOr(attr, "")); v != "" {
				return v
			}
		}
	}
	if name := cleanText(opp.Find(".name").First().Text()); name != "" {
		return name
	}
	if a := opp.Find(".name a").First(); a.Length() > 0 {
		if t := cleanText(a.Text()); t != "" {
			return t
		}
		if title, _ := a.Attr("title"); title != "" {
			return cleanText(title)
		}
	}
	if label := strings.TrimSpace(opp.AttrOr("aria-label", "")); label != "" {
		return strings.ReplaceAll(label, "_", " ")
	}
	return ""
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
		if dyn := opp.Find(".team-name-dynamic").First(); dyn.Length() > 0 {
			for _, attr := range []string{"data-team-bracketname", "data-team-shortname", "data-team-name"} {
				if v := cleanText(dyn.AttrOr(attr, "")); v != "" {
					name = v
					break
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
	return participantFromIdentity(name, link, race)
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
	return participantFromIdentity(name, link, race)
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

func bracketStage(match *goquery.Selection, h2, h3, h4 string) string {
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
	if h3 != "" || h4 != "" {
		return joinStage(h3, h4, round)
	}
	return joinStage(h2, round)
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
