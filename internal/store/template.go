package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrDuplicateName = errors.New("duplicate name")

type AnnouncementTemplate struct {
	ID        int64     `json:"id"`
	GuildID   string    `json:"guildId"`
	Name      string    `json:"name"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (s *Store) ListAnnouncementTemplates(ctx context.Context, guildID string) ([]AnnouncementTemplate, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, guild_id, name, title, body, updated_at
		FROM announcement_templates WHERE guild_id = $1
		ORDER BY lower(name)
		LIMIT 50`, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AnnouncementTemplate{}
	for rows.Next() {
		item, err := scanAnnouncementTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) GetAnnouncementTemplate(ctx context.Context, guildID string, id int64) (AnnouncementTemplate, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, guild_id, name, title, body, updated_at
		FROM announcement_templates WHERE guild_id = $1 AND id = $2`, guildID, id)
	item, err := scanAnnouncementTemplate(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return AnnouncementTemplate{}, ErrNotFound
	}
	return item, err
}

func (s *Store) CreateAnnouncementTemplate(ctx context.Context, item AnnouncementTemplate) (AnnouncementTemplate, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO announcement_templates (guild_id, name, title, body)
		VALUES ($1, $2, $3, $4)
		RETURNING id, updated_at`,
		item.GuildID, item.Name, item.Title, item.Body,
	).Scan(&item.ID, &item.UpdatedAt)
	if isUniqueViolation(err) {
		return AnnouncementTemplate{}, ErrDuplicateName
	}
	return item, err
}

func (s *Store) UpdateAnnouncementTemplate(ctx context.Context, item AnnouncementTemplate) (AnnouncementTemplate, error) {
	err := s.pool.QueryRow(ctx, `
		UPDATE announcement_templates
		SET name = $3, title = $4, body = $5, updated_at = now()
		WHERE guild_id = $1 AND id = $2
		RETURNING updated_at`,
		item.GuildID, item.ID, item.Name, item.Title, item.Body,
	).Scan(&item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return AnnouncementTemplate{}, ErrNotFound
	}
	if isUniqueViolation(err) {
		return AnnouncementTemplate{}, ErrDuplicateName
	}
	return item, err
}

func (s *Store) DeleteAnnouncementTemplate(ctx context.Context, guildID string, id int64) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM announcement_templates WHERE guild_id = $1 AND id = $2`, guildID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanAnnouncementTemplate(row scanner) (AnnouncementTemplate, error) {
	var item AnnouncementTemplate
	err := row.Scan(&item.ID, &item.GuildID, &item.Name, &item.Title, &item.Body, &item.UpdatedAt)
	return item, err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
