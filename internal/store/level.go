package store

import (
	"context"
	"errors"
	"time"

	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/jackc/pgx/v5"
)

var ErrNoLevelBackup = errors.New("There is no XP backup to restore.")

type LevelSettings struct {
	GuildID           string        `json:"guildId"`
	Enabled           bool          `json:"enabled"`
	XPMin             int           `json:"xpMin"`
	XPMax             int           `json:"xpMax"`
	RateCount         int           `json:"rateCount"`
	RateSeconds       int           `json:"rateSeconds"`
	VoiceXP           bool          `json:"voiceXp"`
	Announce          string        `json:"announce"`
	AnnounceChannelID string        `json:"announceChannelId"`
	Message           string        `json:"message"`
	AnnounceMinLevel  int           `json:"announceMinLevel"`
	AnnounceMultiple  int           `json:"announceMultiple"`
	RewardOnly        bool          `json:"rewardOnly"`
	ReplaceRewards    bool          `json:"replaceRewards"`
	IgnoredChannelIDs []string      `json:"ignoredChannelIds"`
	IgnoredRoleIDs    []string      `json:"ignoredRoleIds"`
	Rewards           []LevelReward `json:"rewards"`
	BackupAt          *time.Time    `json:"backupAt,omitempty"`
}

type LevelReward struct {
	Level  int    `json:"level"`
	RoleID string `json:"roleId"`
}

type LevelMember struct {
	UserID   string
	UserName string
	XP       int
}

type LevelStanding struct {
	UserID   string `json:"userId"`
	UserName string `json:"userName"`
	XP       int    `json:"xp"`
	Level    int    `json:"level"`
	Rank     int    `json:"rank"`
}

type LevelEvent struct {
	UserID    string    `json:"userId"`
	UserName  string    `json:"userName"`
	Level     int       `json:"level"`
	CreatedAt time.Time `json:"createdAt"`
}

func defaultLevels(guildID string) LevelSettings {
	return LevelSettings{
		GuildID:           guildID,
		XPMin:             domain.DefaultXPMin,
		XPMax:             domain.DefaultXPMax,
		RateCount:         domain.DefaultRateCount,
		RateSeconds:       domain.DefaultRateWindow,
		Announce:          domain.AnnounceCurrent,
		IgnoredChannelIDs: []string{},
		IgnoredRoleIDs:    []string{},
		Rewards:           []LevelReward{},
	}
}

func (s *Store) GetLevels(ctx context.Context, guildID string) (LevelSettings, error) {
	cfg := defaultLevels(guildID)
	var channels, roles []byte
	err := s.pool.QueryRow(ctx, `
		SELECT enabled, xp_min, xp_max, rate_count, rate_seconds, voice_xp, announce,
			announce_channel_id, message, announce_min_level, announce_multiple,
			reward_only, replace_rewards, ignored_channel_ids, ignored_role_ids, backup_at
		FROM level_settings WHERE guild_id = $1`, guildID).Scan(
		&cfg.Enabled, &cfg.XPMin, &cfg.XPMax, &cfg.RateCount, &cfg.RateSeconds, &cfg.VoiceXP, &cfg.Announce,
		&cfg.AnnounceChannelID, &cfg.Message, &cfg.AnnounceMinLevel, &cfg.AnnounceMultiple,
		&cfg.RewardOnly, &cfg.ReplaceRewards, &channels, &roles, &cfg.BackupAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return cfg, nil
	}
	if err != nil {
		return LevelSettings{}, err
	}
	cfg.IgnoredChannelIDs = decodeIDs(channels)
	cfg.IgnoredRoleIDs = decodeIDs(roles)
	rewards, err := s.levelRewards(ctx, guildID)
	if err != nil {
		return LevelSettings{}, err
	}
	cfg.Rewards = rewards
	return cfg, nil
}

func (s *Store) levelRewards(ctx context.Context, guildID string) ([]LevelReward, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT level, role_id FROM level_rewards WHERE guild_id = $1 ORDER BY level`, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	rewards := []LevelReward{}
	for rows.Next() {
		var reward LevelReward
		if err := rows.Scan(&reward.Level, &reward.RoleID); err != nil {
			return nil, err
		}
		rewards = append(rewards, reward)
	}
	return rewards, rows.Err()
}

func (s *Store) SaveLevels(ctx context.Context, cfg LevelSettings) error {
	if cfg.IgnoredChannelIDs == nil {
		cfg.IgnoredChannelIDs = []string{}
	}
	if cfg.IgnoredRoleIDs == nil {
		cfg.IgnoredRoleIDs = []string{}
	}
	if cfg.Rewards == nil {
		cfg.Rewards = []LevelReward{}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		INSERT INTO level_settings (
			guild_id, enabled, xp_min, xp_max, rate_count, rate_seconds, voice_xp,
			announce, announce_channel_id, message, announce_min_level, announce_multiple,
			reward_only, replace_rewards, ignored_channel_ids, ignored_role_ids
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (guild_id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			xp_min = EXCLUDED.xp_min,
			xp_max = EXCLUDED.xp_max,
			rate_count = EXCLUDED.rate_count,
			rate_seconds = EXCLUDED.rate_seconds,
			voice_xp = EXCLUDED.voice_xp,
			announce = EXCLUDED.announce,
			announce_channel_id = EXCLUDED.announce_channel_id,
			message = EXCLUDED.message,
			announce_min_level = EXCLUDED.announce_min_level,
			announce_multiple = EXCLUDED.announce_multiple,
			reward_only = EXCLUDED.reward_only,
			replace_rewards = EXCLUDED.replace_rewards,
			ignored_channel_ids = EXCLUDED.ignored_channel_ids,
			ignored_role_ids = EXCLUDED.ignored_role_ids`,
		cfg.GuildID, cfg.Enabled, cfg.XPMin, cfg.XPMax, cfg.RateCount, cfg.RateSeconds, cfg.VoiceXP,
		cfg.Announce, cfg.AnnounceChannelID, cfg.Message, cfg.AnnounceMinLevel, cfg.AnnounceMultiple,
		cfg.RewardOnly, cfg.ReplaceRewards, encodeIDs(cfg.IgnoredChannelIDs), encodeIDs(cfg.IgnoredRoleIDs),
	)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM level_rewards WHERE guild_id = $1`, cfg.GuildID); err != nil {
		return err
	}
	for _, reward := range cfg.Rewards {
		if _, err = tx.Exec(ctx, `
			INSERT INTO level_rewards (guild_id, level, role_id) VALUES ($1, $2, $3)`,
			cfg.GuildID, reward.Level, reward.RoleID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// GainMemberXP adds XP and returns the score before and after the grant.
func (s *Store) GainMemberXP(ctx context.Context, guildID, userID, name string, gain int) (oldXP, newXP int, err error) {
	if gain < 1 {
		return 0, 0, nil
	}
	name = domain.CleanLevelName(name)
	err = s.pool.QueryRow(ctx, `
		WITH prev AS (
			SELECT xp FROM member_levels WHERE guild_id = $1 AND user_id = $2
		),
		up AS (
			INSERT INTO member_levels (guild_id, user_id, user_name, xp)
			VALUES ($1, $2, $3, LEAST($4::int, $5::int))
			ON CONFLICT (guild_id, user_id) DO UPDATE SET
				user_name = CASE WHEN EXCLUDED.user_name <> '' THEN EXCLUDED.user_name ELSE member_levels.user_name END,
				xp = LEAST(member_levels.xp + $4::int, $5::int)
			RETURNING xp
		)
		SELECT COALESCE((SELECT xp FROM prev), 0), (SELECT xp FROM up)`,
		guildID, userID, name, gain, domain.MaxStoredXP(),
	).Scan(&oldXP, &newXP)
	return oldXP, newXP, err
}

// SetMemberXP sets an absolute score and returns the previous score.
func (s *Store) SetMemberXP(ctx context.Context, guildID, userID, name string, xp int) (oldXP, newXP int, err error) {
	xp = domain.ClampXP(xp)
	name = domain.CleanLevelName(name)
	if xp == 0 {
		oldXP, err = s.ClearMemberXP(ctx, guildID, userID)
		return oldXP, 0, err
	}
	err = s.pool.QueryRow(ctx, `
		WITH prev AS (
			SELECT xp FROM member_levels WHERE guild_id = $1 AND user_id = $2
		),
		up AS (
			INSERT INTO member_levels (guild_id, user_id, user_name, xp)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (guild_id, user_id) DO UPDATE SET
				user_name = CASE WHEN EXCLUDED.user_name <> '' THEN EXCLUDED.user_name ELSE member_levels.user_name END,
				xp = EXCLUDED.xp
			RETURNING xp
		)
		SELECT COALESCE((SELECT xp FROM prev), 0), (SELECT xp FROM up)`,
		guildID, userID, name, xp,
	).Scan(&oldXP, &newXP)
	return oldXP, newXP, err
}

func (s *Store) ClearMemberXP(ctx context.Context, guildID, userID string) (int, error) {
	var xp int
	err := s.pool.QueryRow(ctx, `
		DELETE FROM member_levels WHERE guild_id = $1 AND user_id = $2 RETURNING xp`,
		guildID, userID).Scan(&xp)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return xp, err
}

func (s *Store) MemberLevel(ctx context.Context, guildID, userID string) (LevelStanding, bool, error) {
	var row LevelStanding
	err := s.pool.QueryRow(ctx, `
		SELECT user_id, user_name, xp,
			1 + (
				SELECT count(*) FROM member_levels other
				WHERE other.guild_id = member_levels.guild_id
					AND other.xp > 0
					AND (
						other.xp > member_levels.xp
						OR (other.xp = member_levels.xp AND other.user_id < member_levels.user_id)
					)
			)
		FROM member_levels
		WHERE guild_id = $1 AND user_id = $2`, guildID, userID).Scan(&row.UserID, &row.UserName, &row.XP, &row.Rank)
	if errors.Is(err, pgx.ErrNoRows) {
		return LevelStanding{UserID: userID}, false, nil
	}
	if err != nil {
		return LevelStanding{}, false, err
	}
	if row.XP <= 0 {
		row.Rank = 0
	}
	row.Level = domain.LevelForXP(row.XP)
	return row, true, nil
}

func (s *Store) Leaderboard(ctx context.Context, guildID string, page int) ([]LevelStanding, int, error) {
	if page < 1 {
		page = 1
	}
	if page > domain.MaxLevelPage {
		page = domain.MaxLevelPage
	}
	var total int
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM member_levels WHERE guild_id = $1 AND xp > 0`, guildID).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * domain.LevelPageSize
	rows, err := s.pool.Query(ctx, `
		SELECT user_id, user_name, xp
		FROM member_levels
		WHERE guild_id = $1 AND xp > 0
		ORDER BY xp DESC, user_id ASC
		LIMIT $2 OFFSET $3`, guildID, domain.LevelPageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []LevelStanding{}
	rank := offset
	for rows.Next() {
		var row LevelStanding
		if err := rows.Scan(&row.UserID, &row.UserName, &row.XP); err != nil {
			return nil, 0, err
		}
		rank++
		row.Rank = rank
		row.Level = domain.LevelForXP(row.XP)
		out = append(out, row)
	}
	return out, total, rows.Err()
}

func (s *Store) RecordLevelUp(ctx context.Context, guildID, userID, name string, level int) error {
	name = domain.CleanLevelName(name)
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO level_events (guild_id, user_id, user_name, level) VALUES ($1, $2, $3, $4)`,
		guildID, userID, name, level); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `
		DELETE FROM level_events
		WHERE guild_id = $1
			AND id < COALESCE((
				SELECT id FROM level_events WHERE guild_id = $1 ORDER BY id DESC OFFSET 99 LIMIT 1
			), 0)`, guildID)
	return err
}

func (s *Store) ListLevelEvents(ctx context.Context, guildID string) ([]LevelEvent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT user_id, user_name, level, created_at
		FROM level_events WHERE guild_id = $1
		ORDER BY id DESC LIMIT 25`, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LevelEvent{}
	for rows.Next() {
		var event LevelEvent
		if err := rows.Scan(&event.UserID, &event.UserName, &event.Level, &event.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, event)
	}
	return out, rows.Err()
}

// BackupAndClearLevels copies every score into the backup, then clears them.
func (s *Store) BackupAndClearLevels(ctx context.Context, guildID string) ([]LevelMember, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `
		SELECT user_id, user_name, xp FROM member_levels WHERE guild_id = $1`, guildID)
	if err != nil {
		return nil, err
	}
	members := []LevelMember{}
	for rows.Next() {
		var member LevelMember
		if err := rows.Scan(&member.UserID, &member.UserName, &member.XP); err != nil {
			rows.Close()
			return nil, err
		}
		members = append(members, member)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM level_backups WHERE guild_id = $1`, guildID); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO level_backups (guild_id, user_id, user_name, xp)
		SELECT guild_id, user_id, user_name, xp FROM member_levels WHERE guild_id = $1`, guildID); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM member_levels WHERE guild_id = $1`, guildID); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM level_events WHERE guild_id = $1`, guildID); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO level_settings (guild_id, backup_at) VALUES ($1, now())
		ON CONFLICT (guild_id) DO UPDATE SET backup_at = now()`, guildID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return members, nil
}

// RestoreLevels replaces current scores with the last server-wide backup.
func (s *Store) RestoreLevels(ctx context.Context, guildID string) ([]LevelMember, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var backupAt *time.Time
	err = tx.QueryRow(ctx, `SELECT backup_at FROM level_settings WHERE guild_id = $1`, guildID).Scan(&backupAt)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && backupAt == nil) {
		return nil, ErrNoLevelBackup
	}
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `
		SELECT user_id, user_name, xp FROM level_backups WHERE guild_id = $1`, guildID)
	if err != nil {
		return nil, err
	}
	members := []LevelMember{}
	for rows.Next() {
		var member LevelMember
		if err := rows.Scan(&member.UserID, &member.UserName, &member.XP); err != nil {
			rows.Close()
			return nil, err
		}
		members = append(members, member)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM member_levels WHERE guild_id = $1`, guildID); err != nil {
		return nil, err
	}
	for _, member := range members {
		if member.XP <= 0 {
			continue
		}
		if _, err = tx.Exec(ctx, `
			INSERT INTO member_levels (guild_id, user_id, user_name, xp) VALUES ($1, $2, $3, $4)`,
			guildID, member.UserID, member.UserName, member.XP); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return members, nil
}
