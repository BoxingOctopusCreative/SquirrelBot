package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrGateUnsaved means the application has not been saved yet.
var ErrGateUnsaved = errors.New("Save the application before posting the message.")

type RoleGate struct {
	GuildID           string   `json:"guildId"`
	Enabled           bool     `json:"enabled"`
	RequiredRoleID    string   `json:"requiredRoleId"`
	DeadlineSeconds   int      `json:"deadlineSeconds"`
	WarnBeforeSeconds int      `json:"warnBeforeSeconds"`
	KickReason        string   `json:"kickReason"`
	DMMessage         string   `json:"dmMessage"`
	LogChannelID      string   `json:"logChannelId"`
	ExemptRoleIDs     []string `json:"exemptRoleIds"`
	Emoji             string   `json:"emoji"`
	ChannelID         string   `json:"channelId"`
	MessageID         string   `json:"messageId"`
	Questions         []string `json:"questions"`
}

func (s *Store) GetGate(ctx context.Context, guildID string) (RoleGate, error) {
	var gate RoleGate
	var raw []byte
	err := s.pool.QueryRow(ctx, `
		SELECT guild_id, enabled, required_role_id, deadline_seconds, warn_before_seconds,
		       kick_reason, dm_message, log_channel_id, exempt_role_ids, emoji, channel_id, message_id
		FROM role_gates WHERE guild_id = $1`, guildID).Scan(
		&gate.GuildID, &gate.Enabled, &gate.RequiredRoleID, &gate.DeadlineSeconds, &gate.WarnBeforeSeconds,
		&gate.KickReason, &gate.DMMessage, &gate.LogChannelID, &raw, &gate.Emoji, &gate.ChannelID, &gate.MessageID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return RoleGate{
			GuildID:           guildID,
			DeadlineSeconds:   86400,
			WarnBeforeSeconds: 3600,
			KickReason:        "Did not finish the server application in time",
			ExemptRoleIDs:     []string{},
			Questions:         []string{},
		}, nil
	}
	if err != nil {
		return RoleGate{}, err
	}
	gate.ExemptRoleIDs = decodeIDs(raw)
	gate.Questions, err = s.gateQuestions(ctx, guildID)
	if err != nil {
		return RoleGate{}, err
	}
	return gate, nil
}

func (s *Store) gateQuestions(ctx context.Context, guildID string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT prompt FROM gate_questions WHERE guild_id = $1 ORDER BY position`, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var prompt string
		if err := rows.Scan(&prompt); err != nil {
			return nil, err
		}
		out = append(out, prompt)
	}
	return out, rows.Err()
}

func (s *Store) SaveGate(ctx context.Context, gate RoleGate) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `
		INSERT INTO role_gates (
			guild_id, enabled, required_role_id, deadline_seconds, warn_before_seconds,
			kick_reason, dm_message, log_channel_id, exempt_role_ids, emoji
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, '')
		ON CONFLICT (guild_id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			required_role_id = EXCLUDED.required_role_id,
			deadline_seconds = EXCLUDED.deadline_seconds,
			warn_before_seconds = EXCLUDED.warn_before_seconds,
			kick_reason = EXCLUDED.kick_reason,
			dm_message = EXCLUDED.dm_message,
			log_channel_id = EXCLUDED.log_channel_id,
			exempt_role_ids = EXCLUDED.exempt_role_ids,
			emoji = ''`,
		gate.GuildID, gate.Enabled, gate.RequiredRoleID, gate.DeadlineSeconds, gate.WarnBeforeSeconds,
		gate.KickReason, gate.DMMessage, gate.LogChannelID, encodeIDs(gate.ExemptRoleIDs),
	); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM gate_questions WHERE guild_id = $1`, gate.GuildID); err != nil {
		return err
	}
	for i, prompt := range gate.Questions {
		if _, err = tx.Exec(ctx, `
			INSERT INTO gate_questions (guild_id, position, prompt) VALUES ($1, $2, $3)`,
			gate.GuildID, i+1, prompt); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) SetGateMessage(ctx context.Context, guildID, channelID, messageID string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE role_gates SET channel_id = $2, message_id = $3 WHERE guild_id = $1`,
		guildID, channelID, messageID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrGateUnsaved
	}
	return nil
}

type GateMember struct {
	GuildID    string     `json:"guildId"`
	UserID     string     `json:"userId"`
	UserName   string     `json:"userName"`
	JoinedAt   time.Time  `json:"joinedAt"`
	DeadlineAt time.Time  `json:"deadlineAt"`
	WarnedAt   *time.Time `json:"warnedAt,omitempty"`
	Status     string     `json:"status"`
}

func (s *Store) UpsertGateMember(ctx context.Context, member GateMember) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO gate_members (guild_id, user_id, user_name, joined_at, deadline_at, warned_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (guild_id, user_id) DO UPDATE SET
			user_name = EXCLUDED.user_name,
			joined_at = EXCLUDED.joined_at,
			deadline_at = EXCLUDED.deadline_at,
			warned_at = EXCLUDED.warned_at,
			status = EXCLUDED.status,
			updated_at = now()`,
		member.GuildID, member.UserID, member.UserName, member.JoinedAt, member.DeadlineAt, member.WarnedAt, member.Status,
	)
	return err
}

func (s *Store) SetGateStatus(ctx context.Context, guildID, userID, status string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE gate_members SET status = $3, updated_at = now()
		WHERE guild_id = $1 AND user_id = $2`, guildID, userID, status)
	return err
}

func (s *Store) MarkGateWarned(ctx context.Context, guildID, userID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE gate_members SET warned_at = now(), updated_at = now()
		WHERE guild_id = $1 AND user_id = $2`, guildID, userID)
	return err
}

func (s *Store) ListGateMembers(ctx context.Context, guildID string) (pending []GateMember, recent []GateMember, err error) {
	pending, err = s.gateMembers(ctx, `
		SELECT guild_id, user_id, user_name, joined_at, deadline_at, warned_at, status
		FROM gate_members
		WHERE guild_id = $1 AND status IN ('pending', 'kick_failed')
		ORDER BY deadline_at ASC
		LIMIT 100`, guildID)
	if err != nil {
		return nil, nil, err
	}
	recent, err = s.gateMembers(ctx, `
		SELECT guild_id, user_id, user_name, joined_at, deadline_at, warned_at, status
		FROM gate_members
		WHERE guild_id = $1 AND status = 'kicked' AND updated_at > now() - interval '7 days'
		ORDER BY updated_at DESC
		LIMIT 20`, guildID)
	return pending, recent, err
}

func (s *Store) gateMembers(ctx context.Context, query, guildID string) ([]GateMember, error) {
	rows, err := s.pool.Query(ctx, query, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []GateMember{}
	for rows.Next() {
		var member GateMember
		if err := rows.Scan(&member.GuildID, &member.UserID, &member.UserName, &member.JoinedAt, &member.DeadlineAt, &member.WarnedAt, &member.Status); err != nil {
			return nil, err
		}
		out = append(out, member)
	}
	return out, rows.Err()
}

type DueGateMember struct {
	GateMember
	RequiredRoleID string
	ExemptRoleIDs  []string
	KickReason     string
	DMMessage      string
	LogChannelID   string
	WarnBefore     int
	DeadlineSecs   int
}

func (s *Store) ClaimDueKicks(ctx context.Context, limit int) ([]DueGateMember, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `
		SELECT gm.guild_id, gm.user_id, gm.user_name, gm.joined_at, gm.deadline_at, gm.warned_at, gm.status,
		       rg.required_role_id, rg.exempt_role_ids, rg.kick_reason, rg.dm_message, rg.log_channel_id,
		       rg.warn_before_seconds, rg.deadline_seconds
		FROM gate_members gm
		JOIN role_gates rg ON rg.guild_id = gm.guild_id
		WHERE gm.status = 'pending' AND rg.enabled AND gm.deadline_at <= now()
		ORDER BY gm.deadline_at
		FOR UPDATE OF gm SKIP LOCKED
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	items, err := scanDue(rows)
	rows.Close()
	if err != nil {
		return nil, err
	}
	for i := range items {
		if _, err := tx.Exec(ctx, `
			UPDATE gate_members SET status = 'kicking', updated_at = now()
			WHERE guild_id = $1 AND user_id = $2`, items[i].GuildID, items[i].UserID); err != nil {
			return nil, err
		}
		items[i].Status = "kicking"
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Store) DueWarnings(ctx context.Context, limit int) ([]DueGateMember, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT gm.guild_id, gm.user_id, gm.user_name, gm.joined_at, gm.deadline_at, gm.warned_at, gm.status,
		       rg.required_role_id, rg.exempt_role_ids, rg.kick_reason, rg.dm_message, rg.log_channel_id,
		       rg.warn_before_seconds, rg.deadline_seconds
		FROM gate_members gm
		JOIN role_gates rg ON rg.guild_id = gm.guild_id
		WHERE gm.status = 'pending'
		  AND gm.warned_at IS NULL
		  AND rg.enabled
		  AND rg.warn_before_seconds > 0
		  AND gm.deadline_at > now()
		  AND gm.deadline_at <= now() + make_interval(secs => rg.warn_before_seconds)
		ORDER BY gm.deadline_at
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDue(rows)
}

func scanDue(rows pgx.Rows) ([]DueGateMember, error) {
	out := []DueGateMember{}
	for rows.Next() {
		var item DueGateMember
		var raw []byte
		if err := rows.Scan(
			&item.GuildID, &item.UserID, &item.UserName, &item.JoinedAt, &item.DeadlineAt, &item.WarnedAt, &item.Status,
			&item.RequiredRoleID, &raw, &item.KickReason, &item.DMMessage, &item.LogChannelID,
			&item.WarnBefore, &item.DeadlineSecs,
		); err != nil {
			return nil, err
		}
		item.ExemptRoleIDs = decodeIDs(raw)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) CountPending(ctx context.Context, guildID string) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM gate_members WHERE guild_id = $1 AND status = 'pending'`, guildID).Scan(&n)
	return n, err
}
