package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type Automod struct {
	GuildID           string   `json:"guildId"`
	Enabled           bool     `json:"enabled"`
	BlockInvites      bool     `json:"blockInvites"`
	BlockSpam         bool     `json:"blockSpam"`
	MentionLimit      int      `json:"mentionLimit"`
	CapsPercent       int      `json:"capsPercent"`
	CapsMinLength     int      `json:"capsMinLength"`
	BadWords          []string `json:"badWords"`
	Action            string   `json:"action"`
	TimeoutSeconds    int      `json:"timeoutSeconds"`
	IgnoredChannelIDs []string `json:"ignoredChannelIds"`
	IgnoredRoleIDs    []string `json:"ignoredRoleIds"`
}

func defaultAutomod(guildID string) Automod {
	return Automod{
		GuildID:           guildID,
		BlockInvites:      true,
		BlockSpam:         true,
		MentionLimit:      5,
		CapsPercent:       70,
		CapsMinLength:     12,
		BadWords:          []string{},
		Action:            "delete",
		TimeoutSeconds:    600,
		IgnoredChannelIDs: []string{},
		IgnoredRoleIDs:    []string{},
	}
}

func (s *Store) GetAutomod(ctx context.Context, guildID string) (Automod, error) {
	var cfg Automod
	var words, channels, roles []byte
	err := s.pool.QueryRow(ctx, `
		SELECT guild_id, enabled, block_invites, block_spam, mention_limit, caps_percent,
		       caps_min_length, bad_words, action, timeout_seconds, ignored_channel_ids, ignored_role_ids
		FROM automod_configs WHERE guild_id = $1`, guildID).Scan(
		&cfg.GuildID, &cfg.Enabled, &cfg.BlockInvites, &cfg.BlockSpam, &cfg.MentionLimit, &cfg.CapsPercent,
		&cfg.CapsMinLength, &words, &cfg.Action, &cfg.TimeoutSeconds, &channels, &roles,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return defaultAutomod(guildID), nil
	}
	if err != nil {
		return Automod{}, err
	}
	cfg.BadWords = decodeIDs(words)
	cfg.IgnoredChannelIDs = decodeIDs(channels)
	cfg.IgnoredRoleIDs = decodeIDs(roles)
	return cfg, nil
}

func (s *Store) SaveAutomod(ctx context.Context, cfg Automod) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO automod_configs (
			guild_id, enabled, block_invites, block_spam, mention_limit, caps_percent,
			caps_min_length, bad_words, action, timeout_seconds, ignored_channel_ids, ignored_role_ids
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (guild_id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			block_invites = EXCLUDED.block_invites,
			block_spam = EXCLUDED.block_spam,
			mention_limit = EXCLUDED.mention_limit,
			caps_percent = EXCLUDED.caps_percent,
			caps_min_length = EXCLUDED.caps_min_length,
			bad_words = EXCLUDED.bad_words,
			action = EXCLUDED.action,
			timeout_seconds = EXCLUDED.timeout_seconds,
			ignored_channel_ids = EXCLUDED.ignored_channel_ids,
			ignored_role_ids = EXCLUDED.ignored_role_ids`,
		cfg.GuildID, cfg.Enabled, cfg.BlockInvites, cfg.BlockSpam, cfg.MentionLimit, cfg.CapsPercent,
		cfg.CapsMinLength, encodeIDs(cfg.BadWords), cfg.Action, cfg.TimeoutSeconds,
		encodeIDs(cfg.IgnoredChannelIDs), encodeIDs(cfg.IgnoredRoleIDs),
	)
	return err
}

type LogConfig struct {
	GuildID       string `json:"guildId"`
	ChannelID     string `json:"channelId"`
	MessageDelete bool   `json:"messageDelete"`
	MessageEdit   bool   `json:"messageEdit"`
	MemberJoin    bool   `json:"memberJoin"`
	MemberLeave   bool   `json:"memberLeave"`
	MemberBan     bool   `json:"memberBan"`
}

func (s *Store) GetLogConfig(ctx context.Context, guildID string) (LogConfig, error) {
	var cfg LogConfig
	err := s.pool.QueryRow(ctx, `
		SELECT guild_id, channel_id, message_delete, message_edit, member_join, member_leave, member_ban
		FROM log_configs WHERE guild_id = $1`, guildID).Scan(
		&cfg.GuildID, &cfg.ChannelID, &cfg.MessageDelete, &cfg.MessageEdit, &cfg.MemberJoin, &cfg.MemberLeave, &cfg.MemberBan,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return LogConfig{
			GuildID: guildID, MessageDelete: true, MessageEdit: true,
			MemberJoin: true, MemberLeave: true, MemberBan: true,
		}, nil
	}
	return cfg, err
}

func (s *Store) SaveLogConfig(ctx context.Context, cfg LogConfig) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO log_configs (
			guild_id, channel_id, message_delete, message_edit, member_join, member_leave, member_ban
		) VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (guild_id) DO UPDATE SET
			channel_id = EXCLUDED.channel_id,
			message_delete = EXCLUDED.message_delete,
			message_edit = EXCLUDED.message_edit,
			member_join = EXCLUDED.member_join,
			member_leave = EXCLUDED.member_leave,
			member_ban = EXCLUDED.member_ban`,
		cfg.GuildID, cfg.ChannelID, cfg.MessageDelete, cfg.MessageEdit, cfg.MemberJoin, cfg.MemberLeave, cfg.MemberBan,
	)
	return err
}

type ReactionRole struct {
	ID        int64  `json:"id"`
	GuildID   string `json:"guildId"`
	ChannelID string `json:"channelId"`
	MessageID string `json:"messageId"`
	Emoji     string `json:"emoji"`
	RoleID    string `json:"roleId"`
}

func scanReaction(row pgx.Row) (ReactionRole, error) {
	var item ReactionRole
	err := row.Scan(&item.ID, &item.GuildID, &item.ChannelID, &item.MessageID, &item.Emoji, &item.RoleID)
	return item, err
}

func (s *Store) ListReactionRoles(ctx context.Context, guildID string) ([]ReactionRole, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, guild_id, channel_id, message_id, emoji, role_id
		FROM reaction_roles WHERE guild_id = $1 ORDER BY id`, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReactionRole{}
	for rows.Next() {
		item, err := scanReaction(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) ReactionRolesForMessage(ctx context.Context, messageID string) ([]ReactionRole, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, guild_id, channel_id, message_id, emoji, role_id
		FROM reaction_roles WHERE message_id = $1 ORDER BY id`, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReactionRole{}
	for rows.Next() {
		item, err := scanReaction(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) HasReactionRole(ctx context.Context, messageID, emoji string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM reaction_roles WHERE message_id = $1 AND emoji = $2)`,
		messageID, emoji).Scan(&exists)
	return exists, err
}

func (s *Store) AddReactionRole(ctx context.Context, item ReactionRole) (ReactionRole, error) {
	return scanReaction(s.pool.QueryRow(ctx, `
		INSERT INTO reaction_roles (guild_id, channel_id, message_id, emoji, role_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (message_id, emoji) DO UPDATE SET
			role_id = EXCLUDED.role_id,
			channel_id = EXCLUDED.channel_id
		RETURNING id, guild_id, channel_id, message_id, emoji, role_id`,
		item.GuildID, item.ChannelID, item.MessageID, item.Emoji, item.RoleID))
}

func (s *Store) SetReactionPanel(ctx context.Context, guildID, channelID, messageID string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE guilds SET reaction_channel_id = $2, reaction_message_id = $3, updated_at = now()
		WHERE id = $1`, guildID, channelID, messageID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) RetargetReactionRoles(ctx context.Context, guildID, channelID, messageID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE reaction_roles SET channel_id = $2, message_id = $3 WHERE guild_id = $1`,
		guildID, channelID, messageID)
	return err
}

func (s *Store) DeleteReactionRole(ctx context.Context, guildID string, id int64) (ReactionRole, error) {
	item, err := scanReaction(s.pool.QueryRow(ctx, `
		DELETE FROM reaction_roles WHERE guild_id = $1 AND id = $2
		RETURNING id, guild_id, channel_id, message_id, emoji, role_id`, guildID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return ReactionRole{}, ErrNotFound
	}
	return item, err
}

type CustomCommand struct {
	GuildID  string   `json:"guildId"`
	Name     string   `json:"name"`
	Response string   `json:"response"`
	Kind     string   `json:"kind"`
	RoleIDs  []string `json:"roleIds"`
}

func scanCommand(guildID, name, response, kind string, roles []string) CustomCommand {
	if roles == nil {
		roles = []string{}
	}
	if kind == "" {
		kind = "text"
	}
	return CustomCommand{GuildID: guildID, Name: name, Response: response, Kind: kind, RoleIDs: roles}
}

func (s *Store) ListCommands(ctx context.Context, guildID string) ([]CustomCommand, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT guild_id, name, response, kind, role_ids FROM custom_commands
		WHERE guild_id = $1 ORDER BY name`, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CustomCommand{}
	for rows.Next() {
		var guild, name, response, kind string
		var roles []string
		if err := rows.Scan(&guild, &name, &response, &kind, &roles); err != nil {
			return nil, err
		}
		out = append(out, scanCommand(guild, name, response, kind, roles))
	}
	return out, rows.Err()
}

func (s *Store) GetCommand(ctx context.Context, guildID, name string) (CustomCommand, error) {
	var guild, commandName, response, kind string
	var roles []string
	err := s.pool.QueryRow(ctx, `
		SELECT guild_id, name, response, kind, role_ids FROM custom_commands
		WHERE guild_id = $1 AND name = $2`, guildID, name).Scan(&guild, &commandName, &response, &kind, &roles)
	if errors.Is(err, pgx.ErrNoRows) {
		return CustomCommand{}, ErrNotFound
	}
	if err != nil {
		return CustomCommand{}, err
	}
	return scanCommand(guild, commandName, response, kind, roles), nil
}

func (s *Store) SaveCommand(ctx context.Context, cmd CustomCommand) error {
	if cmd.RoleIDs == nil {
		cmd.RoleIDs = []string{}
	}
	if cmd.Kind == "" {
		cmd.Kind = "lua"
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO custom_commands (guild_id, name, response, kind, role_ids)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (guild_id, name) DO UPDATE SET
			response = EXCLUDED.response,
			kind = EXCLUDED.kind,
			role_ids = EXCLUDED.role_ids`,
		cmd.GuildID, cmd.Name, cmd.Response, cmd.Kind, cmd.RoleIDs)
	return err
}

func (s *Store) DeleteCommand(ctx context.Context, guildID, name string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM custom_commands WHERE guild_id = $1 AND name = $2`, guildID, name)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type Session struct {
	ID             string
	UserID         string
	Username       string
	Avatar         string
	AccessToken    string
	RefreshToken   string
	TokenExpiresAt time.Time
	ExpiresAt      time.Time
}

func (s *Store) SaveSession(ctx context.Context, session Session) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (
			id, user_id, username, avatar, access_token, refresh_token, token_expires_at, expires_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (id) DO UPDATE SET
			username = EXCLUDED.username,
			avatar = EXCLUDED.avatar,
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			token_expires_at = EXCLUDED.token_expires_at,
			expires_at = EXCLUDED.expires_at`,
		session.ID, session.UserID, session.Username, session.Avatar,
		session.AccessToken, session.RefreshToken, session.TokenExpiresAt, session.ExpiresAt,
	)
	return err
}

func (s *Store) GetSession(ctx context.Context, id string) (Session, error) {
	var session Session
	err := s.pool.QueryRow(ctx, `
		SELECT id, user_id, username, avatar, access_token, refresh_token, token_expires_at, expires_at
		FROM sessions WHERE id = $1 AND expires_at > now()`, id).Scan(
		&session.ID, &session.UserID, &session.Username, &session.Avatar,
		&session.AccessToken, &session.RefreshToken, &session.TokenExpiresAt, &session.ExpiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	return session, err
}

func (s *Store) DeleteSession(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, id)
	return err
}

func (s *Store) DeleteExpiredSessions(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at < now()`)
	return err
}

func (s *Store) RecoverStuckWork(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE announcements
		SET status = 'failed', error = 'The bot restarted before Discord accepted the message.'
		WHERE status = 'sending';
		UPDATE escalations
		SET status = 'expire_failed', end_reason = 'The bot restarted while removing the role.', ended_at = now()
		WHERE status = 'expiring';
		UPDATE gate_members SET status = 'pending', updated_at = now() WHERE status = 'kicking';
	`)
	return err
}
