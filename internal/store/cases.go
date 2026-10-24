package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type ModCase struct {
	ID              int64     `json:"id"`
	GuildID         string    `json:"guildId"`
	CaseNumber      int       `json:"caseNumber"`
	Action          string    `json:"action"`
	TargetID        string    `json:"targetId"`
	TargetName      string    `json:"targetName"`
	ModeratorID     string    `json:"moderatorId"`
	ModeratorName   string    `json:"moderatorName"`
	Reason          string    `json:"reason"`
	DurationSeconds *int      `json:"durationSeconds,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}

type NewCase struct {
	GuildID         string
	Action          string
	TargetID        string
	TargetName      string
	ModeratorID     string
	ModeratorName   string
	Reason          string
	DurationSeconds *int
}

func (s *Store) CreateCase(ctx context.Context, in NewCase) (ModCase, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ModCase{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "case:"+in.GuildID); err != nil {
		return ModCase{}, err
	}
	var number int
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(case_number), 0) + 1 FROM mod_cases WHERE guild_id = $1`, in.GuildID).Scan(&number); err != nil {
		return ModCase{}, err
	}
	var created ModCase
	err = tx.QueryRow(ctx, `
		INSERT INTO mod_cases (
			guild_id, case_number, action, target_id, target_name,
			moderator_id, moderator_name, reason, duration_seconds
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, guild_id, case_number, action, target_id, target_name,
		          moderator_id, moderator_name, reason, duration_seconds, created_at`,
		in.GuildID, number, in.Action, in.TargetID, in.TargetName,
		in.ModeratorID, in.ModeratorName, in.Reason, in.DurationSeconds,
	).Scan(
		&created.ID, &created.GuildID, &created.CaseNumber, &created.Action,
		&created.TargetID, &created.TargetName, &created.ModeratorID, &created.ModeratorName,
		&created.Reason, &created.DurationSeconds, &created.CreatedAt,
	)
	if err != nil {
		return ModCase{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ModCase{}, err
	}
	return created, nil
}

func (s *Store) ListCases(ctx context.Context, guildID string, limit int) ([]ModCase, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, guild_id, case_number, action, target_id, target_name,
		       moderator_id, moderator_name, reason, duration_seconds, created_at
		FROM mod_cases WHERE guild_id = $1
		ORDER BY case_number DESC
		LIMIT $2`, guildID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectCases(rows)
}

func (s *Store) ListCasesForUser(ctx context.Context, guildID, userID string, limit int) ([]ModCase, error) {
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, guild_id, case_number, action, target_id, target_name,
		       moderator_id, moderator_name, reason, duration_seconds, created_at
		FROM mod_cases WHERE guild_id = $1 AND target_id = $2
		ORDER BY case_number DESC
		LIMIT $3`, guildID, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectCases(rows)
}

func collectCases(rows pgx.Rows) ([]ModCase, error) {
	out := []ModCase{}
	for rows.Next() {
		var c ModCase
		if err := rows.Scan(
			&c.ID, &c.GuildID, &c.CaseNumber, &c.Action, &c.TargetID, &c.TargetName,
			&c.ModeratorID, &c.ModeratorName, &c.Reason, &c.DurationSeconds, &c.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) NextPendingKick(ctx context.Context, guildID string) (*time.Time, error) {
	var at time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT deadline_at FROM gate_members
		WHERE guild_id = $1 AND status = 'pending'
		ORDER BY deadline_at ASC LIMIT 1`, guildID).Scan(&at)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &at, nil
}
