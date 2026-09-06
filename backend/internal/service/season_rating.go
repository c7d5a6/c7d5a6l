package service

import (
	"context"

	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/rating"
)

type ratingPlan struct {
	beforeTourIDs []int64
	afterTourIDs  []int64
	flTourID      int64
	applyBefore   bool
	applyFL       bool
}

func (s *Season) resolveRatingPlan(ctx context.Context, active *model.Season, tournamentIDs []int64, closingFantasyLeagueID *int64) (ratingPlan, error) {
	ids := tournamentIDs
	if ids == nil {
		ids, err := s.finishedSeasonTournamentIDs(ctx, active)
		if err != nil {
			return ratingPlan{}, err
		}
		tournamentIDs = ids
	}

	if closingFantasyLeagueID != nil {
		flTourID, err := s.repo.FantasyTournamentFromLeague(ctx, s.db, *closingFantasyLeagueID)
		if err != nil {
			return ratingPlan{}, err
		}
		return ratingPlan{
			beforeTourIDs: tournamentIDs,
			afterTourIDs:  nil,
			flTourID:      flTourID,
			applyBefore:   true,
			applyFL:       true,
		}, nil
	}

	prev, err := s.repo.GetPreviousClosedSeason(ctx, s.db, active.ID)
	if err != nil {
		return ratingPlan{}, err
	}
	if prev != nil && prev.ClosingFantasyLeagueID != nil {
		flTourID, err := s.repo.FantasyTournamentFromLeague(ctx, s.db, *prev.ClosingFantasyLeagueID)
		if err != nil {
			return ratingPlan{}, err
		}
		before, after, err := s.repo.SplitTournamentsAroundFantasy(ctx, s.db, tournamentIDs, flTourID)
		if err != nil {
			return ratingPlan{}, err
		}
		return ratingPlan{
			beforeTourIDs: before,
			afterTourIDs:  after,
			flTourID:      flTourID,
			applyBefore:   false,
			applyFL:       false,
		}, nil
	}

	return ratingPlan{
		beforeTourIDs: tournamentIDs,
		afterTourIDs:  nil,
		flTourID:      0,
		applyBefore:   true,
		applyFL:       false,
	}, nil
}

func (s *Season) loadRatingMatches(ctx context.Context, plan ratingPlan) (before, fl, after []rating.Match, err error) {
	if plan.applyBefore && len(plan.beforeTourIDs) > 0 {
		before, err = s.repo.ListRatingMatches(ctx, s.db, plan.beforeTourIDs)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if plan.applyFL && plan.flTourID > 0 {
		fl, err = s.repo.ListRatingMatches(ctx, s.db, []int64{plan.flTourID})
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if len(plan.afterTourIDs) > 0 {
		after, err = s.repo.ListRatingMatches(ctx, s.db, plan.afterTourIDs)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	return before, fl, after, nil
}
