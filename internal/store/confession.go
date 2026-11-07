package store

import (
	"context"
	"errors"

	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/jackc/pgx/v5"
)

// ErrConfessionClosed means a moderator already approved or rejected it.
var ErrConfessionClosed = errors.New("that confession was already reviewed")

type ConfessionConfig struct {
	GuildID         string `json:"guildId"`
	ReviewChannelID string `json:"reviewChannelId"`
	ChannelID       string `json:"channelId"`
}

type Confession struct {
	ID              int64
	GuildID         string
	AuthorID        string
	AuthorName      string
	Body            string
	Status          string
	ReviewChannelID string
	ReviewMessageID string
	PublicChannelID string
	PublicMessageID string
	ReviewerID      string
	ReviewerName    string
}

func (s *Store) GetConfessionConfig(ctx context.Context, guildID string) (ConfessionConfig, error) {
	var cfg ConfessionConfig
	err := s.pool.QueryRow(ctx, `
		SELECT guild_id, review_channel_id, channel_id
		FROM confession_configs WHERE guild_id = $1`, guildID).Scan(
		&cfg.GuildID, &cfg.ReviewChannelID, &cfg.ChannelID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ConfessionConfig{GuildID: guildID}, nil
	}
	return cfg, err
}

func (s *Store) SaveConfessionConfig(ctx context.Context, cfg ConfessionConfig) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO confession_configs (guild_id, review_channel_id, channel_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (guild_id) DO UPDATE SET
			review_channel_id = EXCLUDED.review_channel_id,
			channel_id = EXCLUDED.channel_id`,
		cfg.GuildID, cfg.ReviewChannelID, cfg.ChannelID,
	)
	return err
}

func (s *Store) CountPendingConfessions(ctx context.Context, guildID, authorID string) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM confessions
		WHERE guild_id = $1 AND author_id = $2 AND status = 'pending'`, guildID, authorID).Scan(&n)
	return n, err
}

func scanConfession(row pgx.Row) (Confession, error) {
	var item Confession
	err := row.Scan(
		&item.ID, &item.GuildID, &item.AuthorID, &item.AuthorName, &item.Body, &item.Status,
		&item.ReviewChannelID, &item.ReviewMessageID, &item.PublicChannelID, &item.PublicMessageID,
		&item.ReviewerID, &item.ReviewerName,
	)
	return item, err
}

func (s *Store) CreateConfession(ctx context.Context, item Confession) (Confession, error) {
	return scanConfession(s.pool.QueryRow(ctx, `
		INSERT INTO confessions (guild_id, author_id, author_name, body, status)
		VALUES ($1, $2, $3, $4, 'pending')
		RETURNING id, guild_id, author_id, author_name, body, status,
		          review_channel_id, review_message_id, public_channel_id, public_message_id,
		          reviewer_id, reviewer_name`,
		item.GuildID, item.AuthorID, item.AuthorName, item.Body,
	))
}

func (s *Store) AttachConfessionReview(ctx context.Context, id int64, channelID, messageID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE confessions SET review_channel_id = $2, review_message_id = $3 WHERE id = $1`,
		id, channelID, messageID,
	)
	return err
}

func (s *Store) DeleteConfession(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM confessions WHERE id = $1`, id)
	return err
}

func (s *Store) GetConfession(ctx context.Context, guildID string, id int64) (Confession, error) {
	item, err := scanConfession(s.pool.QueryRow(ctx, `
		SELECT id, guild_id, author_id, author_name, body, status,
		       review_channel_id, review_message_id, public_channel_id, public_message_id,
		       reviewer_id, reviewer_name
		FROM confessions WHERE id = $1 AND guild_id = $2`, id, guildID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Confession{}, ErrNotFound
	}
	return item, err
}

// ClaimConfession moves a pending confession to approved or rejected.
// A second moderator gets ErrConfessionClosed.
func (s *Store) ClaimConfession(ctx context.Context, guildID string, id int64, status, reviewerID, reviewerName, publicChannelID string) (Confession, error) {
	if status != domain.ConfessionApproved && status != domain.ConfessionRejected {
		return Confession{}, errors.New("choose approve or reject")
	}
	item, err := scanConfession(s.pool.QueryRow(ctx, `
		UPDATE confessions SET
			status = $3,
			reviewer_id = $4,
			reviewer_name = $5,
			public_channel_id = $6,
			decided_at = now()
		WHERE id = $1 AND guild_id = $2 AND status = 'pending'
		RETURNING id, guild_id, author_id, author_name, body, status,
		          review_channel_id, review_message_id, public_channel_id, public_message_id,
		          reviewer_id, reviewer_name`,
		id, guildID, status, reviewerID, reviewerName, publicChannelID,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return Confession{}, ErrConfessionClosed
	}
	return item, err
}

func (s *Store) SetConfessionPublicMessage(ctx context.Context, id int64, channelID, messageID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE confessions SET public_channel_id = $2, public_message_id = $3 WHERE id = $1`,
		id, channelID, messageID,
	)
	return err
}

// ReleaseConfession puts an approved confession back to pending when the public post failed.
func (s *Store) ReleaseConfession(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE confessions SET
			status = 'pending',
			reviewer_id = '',
			reviewer_name = '',
			public_channel_id = '',
			public_message_id = '',
			decided_at = NULL
		WHERE id = $1 AND public_message_id = ''`, id)
	return err
}
