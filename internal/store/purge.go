package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type ChannelPurge struct {
	GuildID         string
	ChannelID       string
	Preset          string
	IntervalSeconds int
	NextRunAt       *time.Time
	Status          string
	Generation      int64
	CursorID        string
	RemovedCount    int
	SkippedCount    int
	LastRunAt       *time.Time
	LastRemoved     int
	LastSkipped     int
	Error           string
}

const purgeSelect = `guild_id, channel_id, preset, interval_seconds, next_run_at, status, generation,
	cursor_id, removed_count, skipped_count, last_run_at, last_removed, last_skipped, error`

func scanPurge(row scanner) (ChannelPurge, error) {
	var item ChannelPurge
	err := row.Scan(
		&item.GuildID, &item.ChannelID, &item.Preset, &item.IntervalSeconds, &item.NextRunAt, &item.Status, &item.Generation,
		&item.CursorID, &item.RemovedCount, &item.SkippedCount, &item.LastRunAt, &item.LastRemoved, &item.LastSkipped, &item.Error,
	)
	return item, err
}

func (s *Store) ListChannelPurges(ctx context.Context, guildID string) ([]ChannelPurge, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+purgeSelect+` FROM channel_purges WHERE guild_id = $1 ORDER BY channel_id`, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ChannelPurge{}
	for rows.Next() {
		item, err := scanPurge(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) GetChannelPurge(ctx context.Context, guildID, channelID string) (ChannelPurge, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+purgeSelect+` FROM channel_purges WHERE guild_id = $1 AND channel_id = $2`, guildID, channelID)
	item, err := scanPurge(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return ChannelPurge{}, ErrNotFound
	}
	return item, err
}

func (s *Store) SaveChannelPurge(ctx context.Context, guildID, channelID, preset string, seconds int) (ChannelPurge, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO channel_purges (guild_id, channel_id, preset, interval_seconds, next_run_at, status, generation)
		VALUES ($1, $2, $3, $4::int, now() + ($4::int * interval '1 second'), 'scheduled', 1)
		ON CONFLICT (guild_id, channel_id) DO UPDATE SET
			preset = EXCLUDED.preset,
			interval_seconds = EXCLUDED.interval_seconds,
			next_run_at = now() + (EXCLUDED.interval_seconds * interval '1 second'),
			status = 'scheduled',
			generation = channel_purges.generation + 1,
			cursor_id = '',
			removed_count = 0,
			skipped_count = 0,
			error = '',
			updated_at = now()
		RETURNING `+purgeSelect,
		guildID, channelID, preset, seconds)
	return scanPurge(row)
}

func (s *Store) DeleteChannelPurge(ctx context.Context, guildID, channelID string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM channel_purges WHERE guild_id = $1 AND channel_id = $2`, guildID, channelID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) SetPurgeEnabled(ctx context.Context, guildID string, on bool) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE guilds SET purge_enabled = $2, updated_at = now() WHERE id = $1`, guildID, on)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if on {
		if _, err := tx.Exec(ctx, `
			UPDATE channel_purges
			SET next_run_at = now() + (interval_seconds * interval '1 second'), updated_at = now()
			WHERE guild_id = $1 AND status = 'scheduled'`, guildID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) RunningPurge(ctx context.Context) (ChannelPurge, bool, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+purgeSelect+` FROM channel_purges WHERE status = 'running' ORDER BY updated_at LIMIT 1`)
	item, err := scanPurge(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return ChannelPurge{}, false, nil
	}
	if err != nil {
		return ChannelPurge{}, false, err
	}
	return item, true, nil
}

func (s *Store) ClaimDuePurge(ctx context.Context) (ChannelPurge, bool, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE channel_purges SET status = 'running', generation = generation + 1, updated_at = now()
		WHERE (guild_id, channel_id) IN (
			SELECT channel_purges.guild_id, channel_purges.channel_id
			FROM channel_purges
			JOIN guilds ON guilds.id = channel_purges.guild_id
			WHERE channel_purges.status = 'scheduled'
				AND channel_purges.next_run_at IS NOT NULL
				AND channel_purges.next_run_at <= now()
				AND guilds.purge_enabled
				AND NOT EXISTS (SELECT 1 FROM channel_purges running WHERE running.status = 'running')
			ORDER BY channel_purges.next_run_at, channel_purges.channel_id
			FOR UPDATE OF channel_purges SKIP LOCKED
			LIMIT 1
		)
		RETURNING `+purgeSelect)
	item, err := scanPurge(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return ChannelPurge{}, false, nil
	}
	if err != nil {
		return ChannelPurge{}, false, err
	}
	return item, true, nil
}

func (s *Store) SavePurgeProgress(ctx context.Context, guildID, channelID string, generation int64, cursor string, removed, skipped int) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE channel_purges
		SET cursor_id = $4, removed_count = $5, skipped_count = $6, updated_at = now()
		WHERE guild_id = $1 AND channel_id = $2 AND generation = $3 AND status = 'running'`,
		guildID, channelID, generation, cursor, removed, skipped)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (s *Store) FinishPurge(ctx context.Context, guildID, channelID string, generation int64, removed, skipped int) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE channel_purges SET
			status = 'scheduled',
			cursor_id = '',
			removed_count = 0,
			skipped_count = 0,
			last_run_at = now(),
			last_removed = $4,
			last_skipped = $5,
			error = '',
			next_run_at = now() + (interval_seconds * interval '1 second'),
			updated_at = now()
		WHERE guild_id = $1 AND channel_id = $2 AND generation = $3 AND status = 'running'`,
		guildID, channelID, generation, removed, skipped)
	return err
}

func (s *Store) FailPurge(ctx context.Context, guildID, channelID string, generation int64, removed, skipped int, reason string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE channel_purges SET
			status = 'scheduled',
			cursor_id = '',
			removed_count = 0,
			skipped_count = 0,
			last_run_at = now(),
			last_removed = $4,
			last_skipped = $5,
			error = $6,
			next_run_at = now() + (interval_seconds * interval '1 second'),
			updated_at = now()
		WHERE guild_id = $1 AND channel_id = $2 AND generation = $3 AND status = 'running'`,
		guildID, channelID, generation, removed, skipped, reason)
	return err
}

func (s *Store) MissPurge(ctx context.Context, guildID, channelID string, generation int64, removed, skipped int) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE channel_purges SET
			status = 'missing',
			cursor_id = '',
			removed_count = 0,
			skipped_count = 0,
			last_run_at = now(),
			last_removed = $4,
			last_skipped = $5,
			error = 'That channel is gone.',
			next_run_at = NULL,
			updated_at = now()
		WHERE guild_id = $1 AND channel_id = $2 AND generation = $3 AND status = 'running'`,
		guildID, channelID, generation, removed, skipped)
	return err
}
