package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// PlayerIdentity is a player row plus alias strings for merge scoring.
type PlayerIdentity struct {
	ID       int64
	Link     string
	Name     string
	RealName string
	Aliases  []string
}

// GetIdentityByID loads one player identity.
func (r *Player) GetIdentityByID(ctx context.Context, q DBTX, playerID int64) (*PlayerIdentity, error) {
	var (
		link     string
		name     sql.NullString
		realName sql.NullString
	)
	err := q.QueryRowContext(ctx, `
		SELECT link, name, real_name FROM player WHERE id = ?
	`, playerID).Scan(&link, &name, &realName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get player identity: %w", err)
	}
	aliases, err := r.listAllAliasNames(ctx, q, playerID)
	if err != nil {
		return nil, err
	}
	return &PlayerIdentity{
		ID:       playerID,
		Link:     link,
		Name:     strings.TrimSpace(name.String),
		RealName: strings.TrimSpace(realName.String),
		Aliases:  aliases,
	}, nil
}

// ListIdentitiesExcept returns every player except excludeID.
func (r *Player) ListIdentitiesExcept(ctx context.Context, q DBTX, excludeID int64) ([]PlayerIdentity, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT id, link, name, real_name FROM player WHERE id != ? ORDER BY name COLLATE NOCASE
	`, excludeID)
	if err != nil {
		return nil, fmt.Errorf("list player identities: %w", err)
	}

	pending := make([]PlayerIdentity, 0)
	for rows.Next() {
		var (
			id       int64
			link     string
			name     sql.NullString
			realName sql.NullString
		)
		if err := rows.Scan(&id, &link, &name, &realName); err != nil {
			rows.Close()
			return nil, err
		}
		pending = append(pending, PlayerIdentity{
			ID:       id,
			Link:     link,
			Name:     strings.TrimSpace(name.String),
			RealName: strings.TrimSpace(realName.String),
		})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	for i := range pending {
		aliases, err := r.listAllAliasNames(ctx, q, pending[i].ID)
		if err != nil {
			return nil, err
		}
		pending[i].Aliases = aliases
	}
	return pending, nil
}

func (r *Player) listAllAliasNames(ctx context.Context, q DBTX, playerID int64) ([]string, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT name FROM player_alias WHERE player_id = ? ORDER BY name COLLATE NOCASE
	`, playerID)
	if err != nil {
		return nil, fmt.Errorf("list alias names: %w", err)
	}
	defer rows.Close()

	out := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

type aliasRow struct {
	ID   int64
	Name string
}

func (r *Player) listAliasRows(ctx context.Context, q DBTX, playerID int64) ([]aliasRow, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT id, name FROM player_alias WHERE player_id = ?
	`, playerID)
	if err != nil {
		return nil, fmt.Errorf("list alias rows: %w", err)
	}
	defer rows.Close()

	out := make([]aliasRow, 0)
	for rows.Next() {
		var row aliasRow
		if err := rows.Scan(&row.ID, &row.Name); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

type raceRow struct {
	ID   int64
	Race string
}

func (r *Player) listRaceRows(ctx context.Context, q DBTX, playerID int64) ([]raceRow, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT id, race FROM player_race WHERE player_id = ? ORDER BY race
	`, playerID)
	if err != nil {
		return nil, fmt.Errorf("list race rows: %w", err)
	}
	defer rows.Close()

	out := make([]raceRow, 0)
	for rows.Next() {
		var row raceRow
		if err := rows.Scan(&row.ID, &row.Race); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Player) raceIDForPlayer(ctx context.Context, q DBTX, playerID int64, race string) (int64, bool, error) {
	var id int64
	err := q.QueryRowContext(ctx, `
		SELECT id FROM player_race WHERE player_id = ? AND race = ?
	`, playerID, race).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("lookup player race: %w", err)
	}
	return id, true, nil
}

// MergePlayers folds aliasPlayerID into mainPlayerID, keeping main elo/race rows.
func (r *Player) MergePlayers(ctx context.Context, q DBTX, mainPlayerID, aliasPlayerID int64) error {
	if mainPlayerID == aliasPlayerID {
		return fmt.Errorf("cannot merge player into itself")
	}
	main, err := r.GetIdentityByID(ctx, q, mainPlayerID)
	if err != nil {
		return err
	}
	if main == nil {
		return fmt.Errorf("main player not found")
	}
	alias, err := r.GetIdentityByID(ctx, q, aliasPlayerID)
	if err != nil {
		return err
	}
	if alias == nil {
		return fmt.Errorf("alias player not found")
	}

	aliasRaces, err := r.listRaceRows(ctx, q, aliasPlayerID)
	if err != nil {
		return err
	}
	for _, ar := range aliasRaces {
		mainRaceID, hasMainRace, err := r.raceIDForPlayer(ctx, q, mainPlayerID, ar.Race)
		if err != nil {
			return err
		}
		if !hasMainRace {
			if _, err := q.ExecContext(ctx, `
				UPDATE player_race SET player_id = ? WHERE id = ?
			`, mainPlayerID, ar.ID); err != nil {
				return fmt.Errorf("move player_race: %w", err)
			}
			continue
		}
		if err := r.repointTournamentPlayers(ctx, q, mainPlayerID, ar.ID, mainRaceID); err != nil {
			return err
		}
		if err := r.repointSeasonSnapshots(ctx, q, ar.ID, mainRaceID); err != nil {
			return err
		}
		if _, err := q.ExecContext(ctx, `DELETE FROM player_race WHERE id = ?`, ar.ID); err != nil {
			return fmt.Errorf("delete merged player_race: %w", err)
		}
	}

	if err := r.repointAliasReferences(ctx, q, mainPlayerID, aliasPlayerID); err != nil {
		return err
	}
	if err := r.absorbAliasNames(ctx, q, mainPlayerID, alias); err != nil {
		return err
	}
	if _, err := q.ExecContext(ctx, `
		DELETE FROM player_import_queue WHERE link = ? COLLATE NOCASE
	`, alias.Link); err != nil {
		return fmt.Errorf("delete alias import queue: %w", err)
	}
	if _, err := q.ExecContext(ctx, `DELETE FROM player WHERE id = ?`, aliasPlayerID); err != nil {
		return fmt.Errorf("delete alias player: %w", err)
	}
	return nil
}

func (r *Player) repointTournamentPlayers(ctx context.Context, q DBTX, mainPlayerID, fromRaceID, toRaceID int64) error {
	rows, err := q.QueryContext(ctx, `
		SELECT id, tournament_id, player_alias_id FROM tournament_player WHERE player_race_id = ?
	`, fromRaceID)
	if err != nil {
		return fmt.Errorf("list tournament_player for race: %w", err)
	}
	defer rows.Close()

	type tpRow struct {
		ID        int64
		TourID    int64
		AliasID   int64
	}
	pending := make([]tpRow, 0)
	for rows.Next() {
		var row tpRow
		if err := rows.Scan(&row.ID, &row.TourID, &row.AliasID); err != nil {
			rows.Close()
			return err
		}
		pending = append(pending, row)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	for _, row := range pending {
		var existingID int64
		err := q.QueryRowContext(ctx, `
			SELECT id FROM tournament_player WHERE tournament_id = ? AND player_race_id = ?
		`, row.TourID, toRaceID).Scan(&existingID)
		switch {
		case err == sql.ErrNoRows:
			newAliasID, err := r.mapAliasID(ctx, q, mainPlayerID, row.AliasID)
			if err != nil {
				return err
			}
			if _, err := q.ExecContext(ctx, `
				UPDATE tournament_player
				SET player_race_id = ?, player_alias_id = ?
				WHERE id = ?
			`, toRaceID, newAliasID, row.ID); err != nil {
				return fmt.Errorf("repoint tournament_player: %w", err)
			}
		case err != nil:
			return fmt.Errorf("lookup tournament_player conflict: %w", err)
		default:
			if err := r.mergeTournamentPlayer(ctx, q, mainPlayerID, row.ID, existingID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Player) mapAliasID(ctx context.Context, q DBTX, mainPlayerID, aliasID int64) (int64, error) {
	var name string
	err := q.QueryRowContext(ctx, `SELECT name FROM player_alias WHERE id = ?`, aliasID).Scan(&name)
	if err == sql.ErrNoRows {
		main, err := r.GetIdentityByID(ctx, q, mainPlayerID)
		if err != nil || main == nil || main.Name == "" {
			return 0, fmt.Errorf("alias row %d missing", aliasID)
		}
		return r.EnsureAliasID(ctx, q, mainPlayerID, main.Name)
	}
	if err != nil {
		return 0, fmt.Errorf("lookup alias name: %w", err)
	}
	return r.EnsureAliasID(ctx, q, mainPlayerID, name)
}

func (r *Player) mergeTournamentPlayer(ctx context.Context, q DBTX, mainPlayerID, fromID, toID int64) error {
	if fromID == toID {
		return nil
	}
	var aliasID int64
	if err := q.QueryRowContext(ctx, `
		SELECT player_alias_id FROM tournament_player WHERE id = ?
	`, fromID).Scan(&aliasID); err != nil {
		return fmt.Errorf("lookup source tournament_player alias: %w", err)
	}
	newAliasID, err := r.mapAliasID(ctx, q, mainPlayerID, aliasID)
	if err != nil {
		return err
	}
	if _, err := q.ExecContext(ctx, `
		UPDATE tournament_player SET player_alias_id = ? WHERE id = ?
	`, newAliasID, toID); err != nil {
		return fmt.Errorf("update target tournament_player alias: %w", err)
	}

	if _, err := q.ExecContext(ctx, `
		UPDATE fantasy_player SET tournament_player_id = ?
		WHERE tournament_player_id = ?
		AND NOT EXISTS (
			SELECT 1 FROM fantasy_player fp
			WHERE fp.fantasy_league_id = fantasy_player.fantasy_league_id
			AND fp.tournament_player_id = ?
		)
	`, toID, fromID, toID); err != nil {
		return fmt.Errorf("repoint fantasy_player: %w", err)
	}
	if _, err := q.ExecContext(ctx, `DELETE FROM fantasy_player WHERE tournament_player_id = ?`, fromID); err != nil {
		return fmt.Errorf("delete duplicate fantasy_player: %w", err)
	}

	if _, err := q.ExecContext(ctx, `
		UPDATE tournament_group_player SET tournament_player_id = ?
		WHERE tournament_player_id = ?
		AND NOT EXISTS (
			SELECT 1 FROM tournament_group_player tgp
			WHERE tgp.tournament_group_id = tournament_group_player.tournament_group_id
			AND tgp.tournament_player_id = ?
		)
	`, toID, fromID, toID); err != nil {
		return fmt.Errorf("repoint tournament_group_player: %w", err)
	}
	if _, err := q.ExecContext(ctx, `DELETE FROM tournament_group_player WHERE tournament_player_id = ?`, fromID); err != nil {
		return fmt.Errorf("delete duplicate tournament_group_player: %w", err)
	}

	if _, err := q.ExecContext(ctx, `
		UPDATE tournament_result SET tournament_player_a_id = ? WHERE tournament_player_a_id = ?
	`, toID, fromID); err != nil {
		return fmt.Errorf("repoint tournament_result a: %w", err)
	}
	if _, err := q.ExecContext(ctx, `
		UPDATE tournament_result SET tournament_player_b_id = ? WHERE tournament_player_b_id = ?
	`, toID, fromID); err != nil {
		return fmt.Errorf("repoint tournament_result b: %w", err)
	}
	if _, err := q.ExecContext(ctx, `DELETE FROM tournament_player WHERE id = ?`, fromID); err != nil {
		return fmt.Errorf("delete merged tournament_player: %w", err)
	}
	return nil
}

func (r *Player) repointSeasonSnapshots(ctx context.Context, q DBTX, fromRaceID, toRaceID int64) error {
	rows, err := q.QueryContext(ctx, `
		SELECT season_id FROM season_player_race WHERE player_race_id = ?
	`, fromRaceID)
	if err != nil {
		return fmt.Errorf("list season snapshots: %w", err)
	}
	defer rows.Close()

	seasons := make([]int64, 0)
	for rows.Next() {
		var seasonID int64
		if err := rows.Scan(&seasonID); err != nil {
			rows.Close()
			return err
		}
		seasons = append(seasons, seasonID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	for _, seasonID := range seasons {
		var n int
		if err := q.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM season_player_race WHERE season_id = ? AND player_race_id = ?
		`, seasonID, toRaceID).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			if _, err := q.ExecContext(ctx, `
				DELETE FROM season_player_race WHERE season_id = ? AND player_race_id = ?
			`, seasonID, fromRaceID); err != nil {
				return fmt.Errorf("delete duplicate season snapshot: %w", err)
			}
			continue
		}
		if _, err := q.ExecContext(ctx, `
			UPDATE season_player_race SET player_race_id = ? WHERE season_id = ? AND player_race_id = ?
		`, toRaceID, seasonID, fromRaceID); err != nil {
			return fmt.Errorf("repoint season snapshot: %w", err)
		}
	}
	return nil
}

func (r *Player) repointAliasReferences(ctx context.Context, q DBTX, mainPlayerID, aliasPlayerID int64) error {
	aliasRows, err := r.listAliasRows(ctx, q, aliasPlayerID)
	if err != nil {
		return err
	}
	for _, row := range aliasRows {
		newID, err := r.EnsureAliasID(ctx, q, mainPlayerID, row.Name)
		if err != nil {
			return err
		}
		if _, err := q.ExecContext(ctx, `
			UPDATE tournament_player SET player_alias_id = ? WHERE player_alias_id = ?
		`, newID, row.ID); err != nil {
			return fmt.Errorf("repoint tournament_player alias: %w", err)
		}
	}
	return nil
}

func (r *Player) absorbAliasNames(ctx context.Context, q DBTX, mainPlayerID int64, alias *PlayerIdentity) error {
	names := make(map[string]string)
	if alias.Name != "" {
		names[strings.ToLower(alias.Name)] = alias.Name
	}
	for _, name := range alias.Aliases {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		names[strings.ToLower(name)] = name
	}
	linkName := linkSlug(alias.Link)
	if linkName != "" {
		names[strings.ToLower(linkName)] = linkName
	}
	for _, name := range names {
		if _, err := r.EnsureAliasID(ctx, q, mainPlayerID, name); err != nil {
			return err
		}
	}
	return nil
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
