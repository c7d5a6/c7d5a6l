package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/c7d5a6/c7d5a6l/internal/model"
)

// DefaultElo is assigned when a new player_race row is created.
const DefaultElo = 1750.0

// PortraitBlob is cached portrait image bytes to store with a player row.
// Nil PortraitBlob on Upsert leaves an existing blob unchanged.
type PortraitBlob struct {
	Data []byte
	Mime string
}

// GetByLink loads a player and alternate IDs (aliases excluding the primary name).
// Returns nil, nil when no row matches. Does not load portrait bytes.
func (r *Player) GetByLink(ctx context.Context, q DBTX, link string) (*model.PlayerPage, error) {
	var (
		id            int64
		name          sql.NullString
		realName      sql.NullString
		preferredRace sql.NullString
		portraitURL   sql.NullString
		hasPortrait   int
	)
	err := q.QueryRowContext(ctx, `
		SELECT id, name, real_name, preferred_race, portrait_url,
			CASE WHEN portrait IS NOT NULL AND length(portrait) > 0 THEN 1 ELSE 0 END
		FROM player
		WHERE link = ? COLLATE NOCASE
	`, link).Scan(&id, &name, &realName, &preferredRace, &portraitURL, &hasPortrait)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get player by link: %w", err)
	}

	page := model.NewPlayerPage(link)
	if name.Valid {
		v := name.String
		page.Name = &v
	}
	if realName.Valid {
		v := realName.String
		page.RealName = &v
	}
	if preferredRace.Valid {
		v := preferredRace.String
		page.PreferredRace = &v
	}
	if portraitURL.Valid {
		v := portraitURL.String
		page.PortraitURL = &v
	}
	page.HasPortrait = hasPortrait == 1

	ids, err := r.listAlternateIDs(ctx, q, id, nullStr(page.Name))
	if err != nil {
		return nil, err
	}
	page.IDs = ids

	raceElos, err := r.listRaceElos(ctx, q, id)
	if err != nil {
		return nil, err
	}
	page.RaceElos = raceElos
	return &page, nil
}

// GetPortraitByLink returns cached portrait bytes for a player link.
// Returns nil, "", nil when the player or portrait is missing.
func (r *Player) GetPortraitByLink(ctx context.Context, q DBTX, link string) ([]byte, string, error) {
	var (
		blob []byte
		mime sql.NullString
	)
	err := q.QueryRowContext(ctx, `
		SELECT portrait, portrait_mime
		FROM player
		WHERE link = ? COLLATE NOCASE
	`, link).Scan(&blob, &mime)
	if err == sql.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("get player portrait: %w", err)
	}
	if len(blob) == 0 {
		return nil, "", nil
	}
	ctype := "application/octet-stream"
	if mime.Valid && strings.TrimSpace(mime.String) != "" {
		ctype = strings.TrimSpace(mime.String)
	}
	return blob, ctype, nil
}

func (r *Player) listAlternateIDs(ctx context.Context, q DBTX, playerID int64, primaryName string) ([]string, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT name FROM player_alias WHERE player_id = ? ORDER BY name COLLATE NOCASE
	`, playerID)
	if err != nil {
		return nil, fmt.Errorf("list aliases: %w", err)
	}
	defer rows.Close()

	primaryKey := strings.ToLower(strings.TrimSpace(primaryName))
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		if primaryKey != "" && strings.ToLower(strings.TrimSpace(name)) == primaryKey {
			continue
		}
		out = append(out, name)
	}
	if out == nil {
		out = []string{}
	}
	return out, rows.Err()
}

func (r *Player) listRaceElos(ctx context.Context, q DBTX, playerID int64) ([]model.RaceElo, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT race, elo FROM player_race
		WHERE player_id = ?
		ORDER BY elo DESC, race ASC
	`, playerID)
	if err != nil {
		return nil, fmt.Errorf("list race elos: %w", err)
	}
	defer rows.Close()

	out := make([]model.RaceElo, 0)
	for rows.Next() {
		var re model.RaceElo
		if err := rows.Scan(&re.Race, &re.Elo); err != nil {
			return nil, err
		}
		out = append(out, re)
	}
	return out, rows.Err()
}

// Upsert inserts or updates the player by link, syncs aliases and preferred race row.
// When portrait is non-nil, portrait bytes/mime are written; nil leaves an existing blob unchanged.
func (r *Player) Upsert(ctx context.Context, q DBTX, page model.PlayerPage, portrait *PortraitBlob) error {
	if page.Link == "" {
		return fmt.Errorf("player link is required")
	}

	var id int64
	err := q.QueryRowContext(ctx, `SELECT id FROM player WHERE link = ? COLLATE NOCASE`, page.Link).Scan(&id)
	switch {
	case err == sql.ErrNoRows:
		if portrait != nil {
			res, err := q.ExecContext(ctx, `
				INSERT INTO player (link, name, real_name, preferred_race, portrait_url, portrait, portrait_mime)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, page.Link, nullableText(page.Name), nullableText(page.RealName), nullableText(page.PreferredRace),
				nullableText(page.PortraitURL), portrait.Data, nullableMime(portrait.Mime))
			if err != nil {
				return fmt.Errorf("insert player: %w", err)
			}
			id, err = res.LastInsertId()
			if err != nil {
				return fmt.Errorf("player last insert id: %w", err)
			}
		} else {
			res, err := q.ExecContext(ctx, `
				INSERT INTO player (link, name, real_name, preferred_race, portrait_url)
				VALUES (?, ?, ?, ?, ?)
			`, page.Link, nullableText(page.Name), nullableText(page.RealName), nullableText(page.PreferredRace), nullableText(page.PortraitURL))
			if err != nil {
				return fmt.Errorf("insert player: %w", err)
			}
			id, err = res.LastInsertId()
			if err != nil {
				return fmt.Errorf("player last insert id: %w", err)
			}
		}
	case err != nil:
		return fmt.Errorf("lookup player: %w", err)
	default:
		if _, err := q.ExecContext(ctx, `
			UPDATE player
			SET name = ?, real_name = ?, preferred_race = ?, portrait_url = ?
			WHERE id = ?
		`, nullableText(page.Name), nullableText(page.RealName), nullableText(page.PreferredRace), nullableText(page.PortraitURL), id); err != nil {
			return fmt.Errorf("update player: %w", err)
		}
		if portrait != nil {
			if _, err := q.ExecContext(ctx, `
				UPDATE player SET portrait = ?, portrait_mime = ? WHERE id = ?
			`, portrait.Data, nullableMime(portrait.Mime), id); err != nil {
				return fmt.Errorf("update player portrait: %w", err)
			}
		}
	}

	if err := r.syncAliases(ctx, q, id, nullStr(page.Name), page.IDs); err != nil {
		return err
	}
	if race := nullStr(page.PreferredRace); race != "" {
		if err := r.ensurePlayerRace(ctx, q, id, race); err != nil {
			return err
		}
	}
	return nil
}

func nullableMime(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}

func (r *Player) syncAliases(ctx context.Context, q DBTX, playerID int64, primaryName string, alternateIDs []string) error {
	want := make(map[string]string) // lower -> display
	if primaryName != "" {
		want[strings.ToLower(primaryName)] = primaryName
	}
	for _, id := range alternateIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		want[strings.ToLower(id)] = id
	}

	rows, err := q.QueryContext(ctx, `SELECT id, name FROM player_alias WHERE player_id = ?`, playerID)
	if err != nil {
		return fmt.Errorf("list aliases for sync: %w", err)
	}
	defer rows.Close()

	existing := make(map[string]int64)
	for rows.Next() {
		var aliasID int64
		var name string
		if err := rows.Scan(&aliasID, &name); err != nil {
			return err
		}
		existing[strings.ToLower(name)] = aliasID
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for key, aliasID := range existing {
		if _, ok := want[key]; ok {
			continue
		}
		if _, err := q.ExecContext(ctx, `DELETE FROM player_alias WHERE id = ?`, aliasID); err != nil {
			return fmt.Errorf("delete alias: %w", err)
		}
	}
	for key, name := range want {
		if _, ok := existing[key]; ok {
			continue
		}
		if _, err := q.ExecContext(ctx, `
			INSERT INTO player_alias (player_id, name) VALUES (?, ?)
		`, playerID, name); err != nil {
			return fmt.Errorf("insert alias: %w", err)
		}
	}
	return nil
}

func (r *Player) ensurePlayerRace(ctx context.Context, q DBTX, playerID int64, race string) error {
	var n int
	err := q.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM player_race WHERE player_id = ? AND race = ?
	`, playerID, race).Scan(&n)
	if err != nil {
		return fmt.Errorf("check player_race: %w", err)
	}
	if n > 0 {
		return nil
	}
	res, err := q.ExecContext(ctx, `
		INSERT INTO player_race (player_id, race, elo) VALUES (?, ?, ?)
	`, playerID, race, DefaultElo)
	if err != nil {
		return fmt.Errorf("insert player_race: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("player_race last insert id: %w", err)
	}
	return ensureActiveSeasonSnapshot(ctx, q, id, DefaultElo)
}

func ensureActiveSeasonSnapshot(ctx context.Context, q DBTX, playerRaceID int64, elo float64) error {
	_, err := q.ExecContext(ctx, `
		INSERT OR IGNORE INTO season_player_race (season_id, player_race_id, start_elo, start_rank)
		SELECT s.id, ?, ?,
			COALESCE((SELECT MAX(spr.start_rank) FROM season_player_race spr WHERE spr.season_id = s.id), 0) + 1
		FROM season s
		WHERE s.status = 'active'
		LIMIT 1
	`, playerRaceID, elo)
	if err != nil {
		return fmt.Errorf("ensure active season snapshot: %w", err)
	}
	return nil
}

// ExistsByLink reports whether a player row exists for link.
func (r *Player) ExistsByLink(ctx context.Context, q DBTX, link string) (bool, error) {
	var n int
	err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM player WHERE link = ? COLLATE NOCASE`, link).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("exists player by link: %w", err)
	}
	return n > 0, nil
}

// IDByLink returns the player id for link, or 0 if missing.
func (r *Player) IDByLink(ctx context.Context, q DBTX, link string) (int64, error) {
	var id int64
	err := q.QueryRowContext(ctx, `SELECT id FROM player WHERE link = ? COLLATE NOCASE`, link).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("player id by link: %w", err)
	}
	return id, nil
}

// EnsureAliasID ensures an alias row exists and returns its id.
func (r *Player) EnsureAliasID(ctx context.Context, q DBTX, playerID int64, alias string) (int64, error) {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return 0, fmt.Errorf("alias name is required")
	}
	var id int64
	err := q.QueryRowContext(ctx, `
		SELECT id FROM player_alias WHERE player_id = ? AND name = ? COLLATE NOCASE
	`, playerID, alias).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("lookup alias: %w", err)
	}
	res, err := q.ExecContext(ctx, `
		INSERT INTO player_alias (player_id, name) VALUES (?, ?)
	`, playerID, alias)
	if err != nil {
		return 0, fmt.Errorf("insert alias: %w", err)
	}
	id, err = res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("alias last insert id: %w", err)
	}
	return id, nil
}

// EnsureRaceID ensures a player_race row exists and returns its id.
func (r *Player) EnsureRaceID(ctx context.Context, q DBTX, playerID int64, race string) (int64, error) {
	race = strings.TrimSpace(strings.ToLower(race))
	if race == "" {
		return 0, fmt.Errorf("race is required")
	}
	var id int64
	err := q.QueryRowContext(ctx, `
		SELECT id FROM player_race WHERE player_id = ? AND race = ?
	`, playerID, race).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("lookup player_race: %w", err)
	}
	res, err := q.ExecContext(ctx, `
		INSERT INTO player_race (player_id, race, elo) VALUES (?, ?, ?)
	`, playerID, race, DefaultElo)
	if err != nil {
		return 0, fmt.Errorf("insert player_race: %w", err)
	}
	id, err = res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("player_race last insert id: %w", err)
	}
	if err := ensureActiveSeasonSnapshot(ctx, q, id, DefaultElo); err != nil {
		return 0, err
	}
	return id, nil
}

// ListRaceEntries returns every player_race row joined with player info, highest elo first.
func (r *Player) ListRaceEntries(ctx context.Context, q DBTX) ([]model.PlayerRaceEntry, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT
			pr.id,
			p.id,
			p.link,
			p.name,
			p.real_name,
			p.preferred_race,
			CASE WHEN p.portrait IS NOT NULL AND length(p.portrait) > 0 THEN 1 ELSE 0 END,
			pr.race,
			pr.elo
		FROM player_race pr
		JOIN player p ON p.id = pr.player_id
		ORDER BY pr.elo DESC, p.name COLLATE NOCASE ASC, pr.race ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list player races: %w", err)
	}
	defer rows.Close()

	out := make([]model.PlayerRaceEntry, 0)
	for rows.Next() {
		var (
			entry         model.PlayerRaceEntry
			name          sql.NullString
			realName      sql.NullString
			preferredRace sql.NullString
			hasPortrait   int
		)
		if err := rows.Scan(
			&entry.PlayerRaceID,
			&entry.PlayerID,
			&entry.Link,
			&name,
			&realName,
			&preferredRace,
			&hasPortrait,
			&entry.Race,
			&entry.Elo,
		); err != nil {
			return nil, fmt.Errorf("scan player race: %w", err)
		}
		if name.Valid {
			v := name.String
			entry.Name = &v
		}
		if realName.Valid {
			v := realName.String
			entry.RealName = &v
		}
		if preferredRace.Valid {
			v := preferredRace.String
			entry.PreferredRace = &v
		}
		entry.HasPortrait = hasPortrait == 1
		out = append(out, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// GetRaceEntryByID returns one player_race row joined with player info.
func (r *Player) GetRaceEntryByID(ctx context.Context, q DBTX, playerRaceID int64) (*model.PlayerRaceEntry, error) {
	row := q.QueryRowContext(ctx, `
		SELECT
			pr.id,
			p.id,
			p.link,
			p.name,
			p.real_name,
			p.preferred_race,
			CASE WHEN p.portrait IS NOT NULL AND length(p.portrait) > 0 THEN 1 ELSE 0 END,
			pr.race,
			pr.elo
		FROM player_race pr
		JOIN player p ON p.id = pr.player_id
		WHERE pr.id = ?
	`, playerRaceID)

	var (
		entry         model.PlayerRaceEntry
		name          sql.NullString
		realName      sql.NullString
		preferredRace sql.NullString
		hasPortrait   int
	)
	err := row.Scan(
		&entry.PlayerRaceID,
		&entry.PlayerID,
		&entry.Link,
		&name,
		&realName,
		&preferredRace,
		&hasPortrait,
		&entry.Race,
		&entry.Elo,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get player race: %w", err)
	}
	if name.Valid {
		v := name.String
		entry.Name = &v
	}
	if realName.Valid {
		v := realName.String
		entry.RealName = &v
	}
	if preferredRace.Valid {
		v := preferredRace.String
		entry.PreferredRace = &v
	}
	entry.HasPortrait = hasPortrait == 1
	return &entry, nil
}

// UpdateRaceElo sets elo for a player_race row. Returns false when the id is missing.
func (r *Player) UpdateRaceElo(ctx context.Context, q DBTX, playerRaceID int64, elo float64) (bool, error) {
	res, err := q.ExecContext(ctx, `
		UPDATE player_race SET elo = ? WHERE id = ?
	`, elo, playerRaceID)
	if err != nil {
		return false, fmt.Errorf("update player race elo: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("update player race elo rows: %w", err)
	}
	return n > 0, nil
}

func nullStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func nullableText(p *string) any {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return s
}
