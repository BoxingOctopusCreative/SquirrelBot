package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type Starboard struct {
	GuildID           string   `json:"guildId"`
	Enabled           bool     `json:"enabled"`
	ChannelID         string   `json:"channelId"`
	Threshold         int      `json:"threshold"`
	SelfStar          bool     `json:"selfStar"`
	NSFW              bool     `json:"nsfw"`
	IgnoredChannelIDs []string `json:"ignoredChannelIds"`
}

type StarPost struct {
	GuildID         string
	SourceChannelID string
	SourceMessageID string
	StarChannelID   string
	StarMessageID   string
}

func (s *Store) GetStarboard(ctx context.Context, guildID string) (Starboard, error) {
	var board Starboard
	var raw []byte
	err := s.pool.QueryRow(ctx, `
		SELECT guild_id, enabled, channel_id, threshold, self_star, nsfw, ignored_channel_ids
		FROM starboards WHERE guild_id = $1`, guildID).Scan(
		&board.GuildID, &board.Enabled, &board.ChannelID, &board.Threshold, &board.SelfStar, &board.NSFW, &raw,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Starboard{GuildID: guildID, Threshold: 3, IgnoredChannelIDs: []string{}}, nil
	}
	if err != nil {
		return Starboard{}, err
	}
	board.IgnoredChannelIDs = decodeIDs(raw)
	return board, nil
}

func (s *Store) SaveStarboard(ctx context.Context, board Starboard) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO starboards (
			guild_id, enabled, channel_id, threshold, self_star, nsfw, ignored_channel_ids
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (guild_id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			channel_id = EXCLUDED.channel_id,
			threshold = EXCLUDED.threshold,
			self_star = EXCLUDED.self_star,
			nsfw = EXCLUDED.nsfw,
			ignored_channel_ids = EXCLUDED.ignored_channel_ids`,
		board.GuildID, board.Enabled, board.ChannelID, board.Threshold, board.SelfStar, board.NSFW, encodeIDs(board.IgnoredChannelIDs),
	)
	return err
}

func (s *Store) StarPost(ctx context.Context, guildID, sourceMessageID string) (StarPost, bool, error) {
	var post StarPost
	err := s.pool.QueryRow(ctx, `
		SELECT guild_id, source_channel_id, source_message_id, star_channel_id, star_message_id
		FROM starboard_posts WHERE guild_id = $1 AND source_message_id = $2`, guildID, sourceMessageID).Scan(
		&post.GuildID, &post.SourceChannelID, &post.SourceMessageID, &post.StarChannelID, &post.StarMessageID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return StarPost{}, false, nil
	}
	if err != nil {
		return StarPost{}, false, err
	}
	return post, true, nil
}

func (s *Store) SaveStarPost(ctx context.Context, post StarPost) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO starboard_posts (
			guild_id, source_channel_id, source_message_id, star_channel_id, star_message_id
		) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (guild_id, source_message_id) DO UPDATE SET
			source_channel_id = EXCLUDED.source_channel_id,
			star_channel_id = EXCLUDED.star_channel_id,
			star_message_id = EXCLUDED.star_message_id`,
		post.GuildID, post.SourceChannelID, post.SourceMessageID, post.StarChannelID, post.StarMessageID,
	)
	return err
}

func (s *Store) DeleteStarPost(ctx context.Context, guildID, sourceMessageID string) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM starboard_posts WHERE guild_id = $1 AND source_message_id = $2`, guildID, sourceMessageID)
	return err
}
