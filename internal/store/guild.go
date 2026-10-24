package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

type Guild struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Icon              string    `json:"icon"`
	ModLogChannelID   string    `json:"modLogChannelId"`
	ModRoleID         string    `json:"modRoleId"`
	AdminRoleID       string    `json:"adminRoleId"`
	WelcomeChannelID  string    `json:"welcomeChannelId"`
	WelcomeMessage    string    `json:"welcomeMessage"`
	AutoroleID        string    `json:"autoroleId"`
	Prefix            string    `json:"prefix"`
	ReactionChannelID string    `json:"reactionChannelId"`
	ReactionMessageID string    `json:"reactionMessageId"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

func (s *Store) EnsureGuild(ctx context.Context, id, name, icon string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO guilds (id, name, icon)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET
			name = CASE WHEN EXCLUDED.name <> '' THEN EXCLUDED.name ELSE guilds.name END,
			icon = CASE WHEN EXCLUDED.icon <> '' THEN EXCLUDED.icon ELSE guilds.icon END,
			updated_at = now()`, id, name, icon)
	return err
}

func (s *Store) GetGuild(ctx context.Context, id string) (Guild, error) {
	var g Guild
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, icon, mod_log_channel_id, mod_role_id, admin_role_id,
		       welcome_channel_id, welcome_message, autorole_id, prefix,
		       reaction_channel_id, reaction_message_id, updated_at
		FROM guilds WHERE id = $1`, id).Scan(
		&g.ID, &g.Name, &g.Icon, &g.ModLogChannelID, &g.ModRoleID, &g.AdminRoleID,
		&g.WelcomeChannelID, &g.WelcomeMessage, &g.AutoroleID, &g.Prefix,
		&g.ReactionChannelID, &g.ReactionMessageID, &g.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Guild{}, ErrNotFound
	}
	return g, err
}

func (s *Store) UpdateGuildSettings(ctx context.Context, g Guild) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE guilds SET
			mod_log_channel_id = $2,
			mod_role_id = $3,
			admin_role_id = $4,
			welcome_channel_id = $5,
			welcome_message = $6,
			autorole_id = $7,
			prefix = $8,
			updated_at = now()
		WHERE id = $1`,
		g.ID, g.ModLogChannelID, g.ModRoleID, g.AdminRoleID,
		g.WelcomeChannelID, g.WelcomeMessage, g.AutoroleID, g.Prefix,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type Counts struct {
	ScheduledAnnouncements int `json:"scheduledAnnouncements"`
	ActiveEscalations      int `json:"activeEscalations"`
	PendingGate            int `json:"pendingGate"`
	Cases                  int `json:"cases"`
}

func (s *Store) Counts(ctx context.Context, guildID string) (Counts, error) {
	var c Counts
	err := s.pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM announcements WHERE guild_id = $1 AND status = 'scheduled'),
			(SELECT count(*) FROM escalations WHERE guild_id = $1 AND status = 'active'),
			(SELECT count(*) FROM gate_members WHERE guild_id = $1 AND status = 'pending'),
			(SELECT count(*) FROM mod_cases WHERE guild_id = $1)`, guildID).Scan(
		&c.ScheduledAnnouncements, &c.ActiveEscalations, &c.PendingGate, &c.Cases,
	)
	return c, err
}

func decodeIDs(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var ids []string
	if err := json.Unmarshal(raw, &ids); err != nil || ids == nil {
		return []string{}
	}
	return ids
}

func encodeIDs(ids []string) []byte {
	if ids == nil {
		ids = []string{}
	}
	raw, err := json.Marshal(ids)
	if err != nil {
		return []byte("[]")
	}
	return raw
}
