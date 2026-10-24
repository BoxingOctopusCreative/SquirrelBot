package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type Announcement struct {
	ID          int64      `json:"id"`
	GuildID     string     `json:"guildId"`
	ChannelID   string     `json:"channelId"`
	AuthorID    string     `json:"authorId"`
	AuthorName  string     `json:"authorName"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	PingRoleID  string     `json:"pingRoleId"`
	ScheduledAt *time.Time `json:"scheduledAt,omitempty"`
	SentAt      *time.Time `json:"sentAt,omitempty"`
	MessageID   string     `json:"messageId"`
	Status      string     `json:"status"`
	Error       string     `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

func (s *Store) CreateAnnouncement(ctx context.Context, a Announcement) (Announcement, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO announcements (
			guild_id, channel_id, author_id, author_name, title, body,
			ping_role_id, scheduled_at, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`,
		a.GuildID, a.ChannelID, a.AuthorID, a.AuthorName, a.Title, a.Body,
		a.PingRoleID, a.ScheduledAt, a.Status,
	).Scan(&a.ID, &a.CreatedAt)
	return a, err
}

func (s *Store) ListAnnouncements(ctx context.Context, guildID string) ([]Announcement, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, guild_id, channel_id, author_id, author_name, title, body,
		       ping_role_id, scheduled_at, sent_at, message_id, status, error, created_at
		FROM announcements WHERE guild_id = $1
		ORDER BY created_at DESC
		LIMIT 50`, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Announcement{}
	for rows.Next() {
		a, err := scanAnnouncement(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) GetAnnouncement(ctx context.Context, guildID string, id int64) (Announcement, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, guild_id, channel_id, author_id, author_name, title, body,
		       ping_role_id, scheduled_at, sent_at, message_id, status, error, created_at
		FROM announcements WHERE guild_id = $1 AND id = $2`, guildID, id)
	a, err := scanAnnouncement(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Announcement{}, ErrNotFound
	}
	return a, err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanAnnouncement(row scanner) (Announcement, error) {
	var a Announcement
	err := row.Scan(
		&a.ID, &a.GuildID, &a.ChannelID, &a.AuthorID, &a.AuthorName, &a.Title, &a.Body,
		&a.PingRoleID, &a.ScheduledAt, &a.SentAt, &a.MessageID, &a.Status, &a.Error, &a.CreatedAt,
	)
	return a, err
}

func (s *Store) DeleteAnnouncement(ctx context.Context, guildID string, id int64) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM announcements
		WHERE guild_id = $1 AND id = $2 AND status IN ('scheduled', 'failed')`, guildID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ClaimDueAnnouncements(ctx context.Context, limit int) ([]Announcement, error) {
	rows, err := s.pool.Query(ctx, `
		UPDATE announcements SET status = 'sending'
		WHERE id IN (
			SELECT id FROM announcements
			WHERE status = 'scheduled' AND scheduled_at IS NOT NULL AND scheduled_at <= now()
			ORDER BY scheduled_at
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		)
		RETURNING id, guild_id, channel_id, author_id, author_name, title, body,
		          ping_role_id, scheduled_at, sent_at, message_id, status, error, created_at`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Announcement{}
	for rows.Next() {
		a, err := scanAnnouncement(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) MarkAnnouncementSent(ctx context.Context, id int64, messageID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE announcements
		SET status = 'sent', message_id = $2, sent_at = now(), error = ''
		WHERE id = $1`, id, messageID)
	return err
}

func (s *Store) MarkAnnouncementFailed(ctx context.Context, id int64, reason string) error {
	if len(reason) > 400 {
		reason = reason[:400]
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE announcements SET status = 'failed', error = $2 WHERE id = $1`, id, reason)
	return err
}

func (s *Store) QueueAnnouncement(ctx context.Context, guildID string, id int64, at time.Time) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE announcements
		SET status = 'scheduled', scheduled_at = $3, error = ''
		WHERE guild_id = $1 AND id = $2 AND status IN ('failed', 'scheduled')`, guildID, id, at)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
