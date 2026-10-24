package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type EscalationRole struct {
	GuildID            string `json:"guildId"`
	RoleID             string `json:"roleId"`
	Tier               string `json:"tier"`
	MaxDurationSeconds int    `json:"maxDurationSeconds"`
}

type Escalation struct {
	ID            int64      `json:"id"`
	GuildID       string     `json:"guildId"`
	UserID        string     `json:"userId"`
	UserName      string     `json:"userName"`
	RoleID        string     `json:"roleId"`
	Tier          string     `json:"tier"`
	GrantedBy     string     `json:"grantedBy"`
	GrantedByName string     `json:"grantedByName"`
	Reason        string     `json:"reason"`
	StartsAt      time.Time  `json:"startsAt"`
	ExpiresAt     time.Time  `json:"expiresAt"`
	EndedAt       *time.Time `json:"endedAt,omitempty"`
	EndReason     string     `json:"endReason,omitempty"`
	Status        string     `json:"status"`
}

func (s *Store) ListEscalationRoles(ctx context.Context, guildID string) ([]EscalationRole, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT guild_id, role_id, tier, max_duration_seconds
		FROM escalation_roles WHERE guild_id = $1 ORDER BY tier, role_id`, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []EscalationRole{}
	for rows.Next() {
		var role EscalationRole
		if err := rows.Scan(&role.GuildID, &role.RoleID, &role.Tier, &role.MaxDurationSeconds); err != nil {
			return nil, err
		}
		out = append(out, role)
	}
	return out, rows.Err()
}

func (s *Store) ReplaceEscalationRoles(ctx context.Context, guildID string, roles []EscalationRole) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM escalation_roles WHERE guild_id = $1`, guildID); err != nil {
		return err
	}
	for _, role := range roles {
		if _, err := tx.Exec(ctx, `
			INSERT INTO escalation_roles (guild_id, role_id, tier, max_duration_seconds)
			VALUES ($1, $2, $3, $4)`, guildID, role.RoleID, role.Tier, role.MaxDurationSeconds); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) EscalationRole(ctx context.Context, guildID, roleID string) (EscalationRole, error) {
	var role EscalationRole
	err := s.pool.QueryRow(ctx, `
		SELECT guild_id, role_id, tier, max_duration_seconds
		FROM escalation_roles WHERE guild_id = $1 AND role_id = $2`, guildID, roleID).Scan(
		&role.GuildID, &role.RoleID, &role.Tier, &role.MaxDurationSeconds,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return EscalationRole{}, ErrNotFound
	}
	return role, err
}

func (s *Store) ActiveEscalation(ctx context.Context, guildID, userID, roleID string) (Escalation, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, guild_id, user_id, user_name, role_id, tier, granted_by, granted_by_name,
		       reason, starts_at, expires_at, ended_at, end_reason, status
		FROM escalations
		WHERE guild_id = $1 AND user_id = $2 AND role_id = $3 AND status = 'active'`, guildID, userID, roleID)
	item, err := scanEscalation(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Escalation{}, ErrNotFound
	}
	return item, err
}

func (s *Store) CreateEscalation(ctx context.Context, item Escalation) (Escalation, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO escalations (
			guild_id, user_id, user_name, role_id, tier, granted_by, granted_by_name,
			reason, expires_at, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'active')
		RETURNING id, starts_at`,
		item.GuildID, item.UserID, item.UserName, item.RoleID, item.Tier,
		item.GrantedBy, item.GrantedByName, item.Reason, item.ExpiresAt,
	).Scan(&item.ID, &item.StartsAt)
	item.Status = "active"
	return item, err
}

func (s *Store) ListEscalations(ctx context.Context, guildID string) ([]Escalation, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, guild_id, user_id, user_name, role_id, tier, granted_by, granted_by_name,
		       reason, starts_at, expires_at, ended_at, end_reason, status
		FROM escalations WHERE guild_id = $1
		ORDER BY CASE WHEN status = 'active' THEN 0 ELSE 1 END, expires_at DESC
		LIMIT 100`, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectEscalations(rows)
}

func (s *Store) GetEscalation(ctx context.Context, guildID string, id int64) (Escalation, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, guild_id, user_id, user_name, role_id, tier, granted_by, granted_by_name,
		       reason, starts_at, expires_at, ended_at, end_reason, status
		FROM escalations WHERE guild_id = $1 AND id = $2`, guildID, id)
	item, err := scanEscalation(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Escalation{}, ErrNotFound
	}
	return item, err
}

func scanEscalation(row scanner) (Escalation, error) {
	var item Escalation
	err := row.Scan(
		&item.ID, &item.GuildID, &item.UserID, &item.UserName, &item.RoleID, &item.Tier,
		&item.GrantedBy, &item.GrantedByName, &item.Reason, &item.StartsAt, &item.ExpiresAt,
		&item.EndedAt, &item.EndReason, &item.Status,
	)
	return item, err
}

func collectEscalations(rows pgx.Rows) ([]Escalation, error) {
	out := []Escalation{}
	for rows.Next() {
		item, err := scanEscalation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) MarkEscalationEnded(ctx context.Context, id int64, status, reason string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE escalations
		SET status = $2, end_reason = $3, ended_at = now()
		WHERE id = $1`, id, status, reason)
	return err
}

func (s *Store) ClaimExpiredEscalations(ctx context.Context, limit int) ([]Escalation, error) {
	rows, err := s.pool.Query(ctx, `
		UPDATE escalations SET status = 'expiring'
		WHERE id IN (
			SELECT id FROM escalations
			WHERE status = 'active' AND expires_at <= now()
			ORDER BY expires_at
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		)
		RETURNING id, guild_id, user_id, user_name, role_id, tier, granted_by, granted_by_name,
		          reason, starts_at, expires_at, ended_at, end_reason, status`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectEscalations(rows)
}
