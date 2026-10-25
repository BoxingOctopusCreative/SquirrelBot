package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type BotProfile struct {
	GuildID    string
	Nick       string
	Avatar     []byte
	AvatarType string
	UpdatedAt  time.Time
}

func (s *Store) GetBotProfile(ctx context.Context, guildID string) (BotProfile, error) {
	profile := BotProfile{GuildID: guildID}
	err := s.pool.QueryRow(ctx, `
		SELECT nick, COALESCE(avatar, ''::bytea), avatar_type, updated_at
		FROM bot_profiles WHERE guild_id = $1`, guildID).Scan(
		&profile.Nick, &profile.Avatar, &profile.AvatarType, &profile.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return profile, nil
	}
	if len(profile.Avatar) == 0 {
		profile.Avatar = nil
		profile.AvatarType = ""
	}
	return profile, err
}

func (s *Store) SaveBotProfile(ctx context.Context, profile BotProfile) error {
	var avatar any
	avatarType := ""
	if len(profile.Avatar) > 0 {
		avatar = profile.Avatar
		avatarType = profile.AvatarType
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO bot_profiles (guild_id, nick, avatar, avatar_type)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (guild_id) DO UPDATE SET
			nick = EXCLUDED.nick,
			avatar = EXCLUDED.avatar,
			avatar_type = EXCLUDED.avatar_type,
			updated_at = now()`,
		profile.GuildID, profile.Nick, avatar, avatarType,
	)
	return err
}
