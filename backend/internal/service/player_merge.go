package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/repository"
)

var (
	ErrPlayerMergeSelf    = fmt.Errorf("%w: cannot merge player into itself", ErrInvalidPlayer)
	ErrPlayerMergeMissing = fmt.Errorf("%w: player not found for merge", ErrPlayerNotFound)
)

// MergeCandidates returns players that may be duplicates of mainPlayerID.
func (s *Player) MergeCandidates(ctx context.Context, mainPlayerID int64, query string, limit int) ([]model.MergeCandidate, error) {
	if mainPlayerID <= 0 {
		return nil, fmt.Errorf("%w: playerId is required", ErrInvalidPlayer)
	}
	if limit <= 0 {
		limit = 15
	}
	if limit > 50 {
		limit = 50
	}

	main, err := s.repo.GetIdentityByID(ctx, s.db, mainPlayerID)
	if err != nil {
		return nil, err
	}
	if main == nil {
		return nil, ErrPlayerMergeMissing
	}

	others, err := s.repo.ListIdentitiesExcept(ctx, s.db, mainPlayerID)
	if err != nil {
		return nil, err
	}

	query = strings.TrimSpace(strings.ToLower(query))
	out := make([]model.MergeCandidate, 0, len(others))
	for _, cand := range others {
		score, reason := scoreMergeCandidate(main, cand, query)
		if score <= 0 {
			continue
		}
		out = append(out, model.MergeCandidate{
			PlayerID:    cand.ID,
			Link:        cand.Link,
			Name:        strPtrOrNil(cand.Name),
			RealName:    strPtrOrNil(cand.RealName),
			Aliases:     cand.Aliases,
			Score:       score,
			MatchReason: reason,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return strings.ToLower(nullStr(out[i].Name)) < strings.ToLower(nullStr(out[j].Name))
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// MergePlayers folds aliasPlayerID into mainPlayerID, keeping main elo and tournament ids.
func (s *Player) MergePlayers(ctx context.Context, mainPlayerID, aliasPlayerID int64) error {
	if mainPlayerID <= 0 || aliasPlayerID <= 0 {
		return fmt.Errorf("%w: mainPlayerId and aliasPlayerId are required", ErrInvalidPlayer)
	}
	if mainPlayerID == aliasPlayerID {
		return ErrPlayerMergeSelf
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := s.repo.MergePlayers(ctx, tx, mainPlayerID, aliasPlayerID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrPlayerMergeMissing
		}
		if strings.Contains(err.Error(), "itself") {
			return ErrPlayerMergeSelf
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	debuglog.Printf("service.Player.MergePlayers main=%d alias=%d", mainPlayerID, aliasPlayerID)
	return nil
}

func scoreMergeCandidate(main *repository.PlayerIdentity, cand repository.PlayerIdentity, query string) (int, string) {
	score := 0
	reasons := make([]string, 0, 4)

	if query != "" {
		if matchesQuery(cand, query) {
			score += 80
			reasons = append(reasons, "matches search")
		} else {
			return 0, ""
		}
	}

	mainTokens := identityTokens(main)
	candTokens := identityTokens(&cand)

	for _, a := range mainTokens {
		for _, b := range candTokens {
			if len(a) < 2 || len(b) < 2 {
				continue
			}
			la, lb := strings.ToLower(a), strings.ToLower(b)
			if la == lb {
				score += 100
				reasons = appendUniqueReason(reasons, "same name/id")
				continue
			}
			if len(la) >= 3 && len(lb) >= 3 && (strings.Contains(la, lb) || strings.Contains(lb, la)) {
				score += 50
				reasons = appendUniqueReason(reasons, "similar name")
			}
		}
	}

	mainSlug := strings.ToLower(linkSlug(main.Link))
	candSlug := strings.ToLower(linkSlug(cand.Link))
	if mainSlug != "" && candSlug != "" {
		if mainSlug == candSlug {
			score += 90
			reasons = appendUniqueReason(reasons, "same link slug")
		} else if len(mainSlug) >= 3 && len(candSlug) >= 3 &&
			(strings.Contains(mainSlug, candSlug) || strings.Contains(candSlug, mainSlug)) {
			score += 40
			reasons = appendUniqueReason(reasons, "similar link")
		}
	}

	if score == 0 {
		return 0, ""
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "possible duplicate")
	}
	return score, strings.Join(reasons, ", ")
}

func identityTokens(p *repository.PlayerIdentity) []string {
	seen := make(map[string]struct{})
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		key := strings.ToLower(s)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
	}
	add(p.Name)
	add(p.RealName)
	add(linkSlug(p.Link))
	for _, alias := range p.Aliases {
		add(alias)
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	return out
}

func matchesQuery(cand repository.PlayerIdentity, query string) bool {
	for _, token := range identityTokens(&cand) {
		if strings.Contains(strings.ToLower(token), query) {
			return true
		}
	}
	return strings.Contains(strings.ToLower(cand.Link), query)
}

func appendUniqueReason(reasons []string, reason string) []string {
	for _, r := range reasons {
		if r == reason {
			return reasons
		}
	}
	return append(reasons, reason)
}

func linkSlug(link string) string {
	link = strings.TrimSpace(link)
	if link == "" {
		return ""
	}
	if i := strings.LastIndex(link, "/"); i >= 0 && i < len(link)-1 {
		return link[i+1:]
	}
	return link
}
