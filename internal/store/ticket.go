package store

import (
	"context"
	"errors"
	"time"

	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/jackc/pgx/v5"
)

const ticketColumns = `id, guild_id, number, opener_id, opener_name, subject, body, status,
	channel_id, control_message_id, claimer_id, claimer_name, closed_by_id, closed_by_name,
	close_reason, transcript, created_at, closed_at`

// TicketConfig is how one server runs support tickets.
type TicketConfig struct {
	GuildID        string `json:"guildId"`
	PanelChannelID string `json:"panelChannelId"`
	PanelMessageID string `json:"-"`
	CategoryID     string `json:"categoryId"`
	SupportRoleID  string `json:"supportRoleId"`
	LogChannelID   string `json:"logChannelId"`
	PanelTitle     string `json:"panelTitle"`
	PanelBody      string `json:"panelBody"`
}

// Ticket is one support request.
type Ticket struct {
	ID               int64      `json:"id"`
	GuildID          string     `json:"-"`
	Number           int        `json:"number"`
	OpenerID         string     `json:"openerId"`
	OpenerName       string     `json:"openerName"`
	Subject          string     `json:"subject"`
	Body             string     `json:"body"`
	Status           string     `json:"status"`
	ChannelID        string     `json:"channelId"`
	ControlMessageID string     `json:"-"`
	ClaimerID        string     `json:"claimerId"`
	ClaimerName      string     `json:"claimerName"`
	ClosedByID       string     `json:"closedById"`
	ClosedByName     string     `json:"closedByName"`
	CloseReason      string     `json:"closeReason"`
	Transcript       string     `json:"transcript"`
	CreatedAt        time.Time  `json:"createdAt"`
	ClosedAt         *time.Time `json:"closedAt,omitempty"`
}

func (s *Store) GetTicketConfig(ctx context.Context, guildID string) (TicketConfig, error) {
	var cfg TicketConfig
	err := s.pool.QueryRow(ctx, `
		SELECT guild_id, panel_channel_id, panel_message_id, category_id, support_role_id,
			log_channel_id, panel_title, panel_body
		FROM ticket_configs WHERE guild_id = $1`, guildID).Scan(
		&cfg.GuildID, &cfg.PanelChannelID, &cfg.PanelMessageID, &cfg.CategoryID, &cfg.SupportRoleID,
		&cfg.LogChannelID, &cfg.PanelTitle, &cfg.PanelBody,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return TicketConfig{
			GuildID: guildID, PanelTitle: domain.DefaultTicketTitle, PanelBody: domain.DefaultTicketBody,
		}, nil
	}
	if err != nil {
		return TicketConfig{}, err
	}
	if cfg.PanelTitle == "" {
		cfg.PanelTitle = domain.DefaultTicketTitle
	}
	if cfg.PanelBody == "" {
		cfg.PanelBody = domain.DefaultTicketBody
	}
	return cfg, nil
}

func (s *Store) SaveTicketConfig(ctx context.Context, cfg TicketConfig) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO ticket_configs (
			guild_id, panel_channel_id, panel_message_id, category_id, support_role_id,
			log_channel_id, panel_title, panel_body
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (guild_id) DO UPDATE SET
			panel_channel_id = EXCLUDED.panel_channel_id,
			panel_message_id = EXCLUDED.panel_message_id,
			category_id = EXCLUDED.category_id,
			support_role_id = EXCLUDED.support_role_id,
			log_channel_id = EXCLUDED.log_channel_id,
			panel_title = EXCLUDED.panel_title,
			panel_body = EXCLUDED.panel_body`,
		cfg.GuildID, cfg.PanelChannelID, cfg.PanelMessageID, cfg.CategoryID, cfg.SupportRoleID,
		cfg.LogChannelID, cfg.PanelTitle, cfg.PanelBody,
	)
	return err
}

func (s *Store) SaveTicketPanelMessage(ctx context.Context, guildID, messageID string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE ticket_configs SET panel_message_id = $2 WHERE guild_id = $1`, guildID, messageID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanTicket(row pgx.Row) (Ticket, error) {
	var item Ticket
	err := row.Scan(
		&item.ID, &item.GuildID, &item.Number, &item.OpenerID, &item.OpenerName, &item.Subject, &item.Body, &item.Status,
		&item.ChannelID, &item.ControlMessageID, &item.ClaimerID, &item.ClaimerName, &item.ClosedByID, &item.ClosedByName,
		&item.CloseReason, &item.Transcript, &item.CreatedAt, &item.ClosedAt,
	)
	return item, err
}

func (s *Store) CreateTicket(ctx context.Context, item Ticket) (Ticket, error) {
	var last error
	for range 3 {
		created, err := s.insertTicket(ctx, item)
		if err == nil {
			return created, nil
		}
		if !isUniqueViolation(err) {
			return Ticket{}, err
		}
		last = err
	}
	if last == nil {
		last = errors.New("could not number that ticket")
	}
	return Ticket{}, last
}

func (s *Store) insertTicket(ctx context.Context, item Ticket) (Ticket, error) {
	return scanTicket(s.pool.QueryRow(ctx, `
		INSERT INTO tickets (guild_id, number, opener_id, opener_name, subject, body)
		VALUES (
			$1,
			(SELECT COALESCE(MAX(number), 0) + 1 FROM tickets WHERE guild_id = $1),
			$2, $3, $4, $5
		)
		RETURNING `+ticketColumns,
		item.GuildID, item.OpenerID, item.OpenerName, item.Subject, item.Body,
	))
}

func (s *Store) AttachTicketChannel(ctx context.Context, guildID string, id int64, channelID, messageID string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE tickets SET channel_id = $3, control_message_id = $4
		WHERE id = $1 AND guild_id = $2`, id, guildID, channelID, messageID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) SetTicketControl(ctx context.Context, guildID string, id int64, messageID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE tickets SET control_message_id = $3 WHERE id = $1 AND guild_id = $2`, id, guildID, messageID)
	return err
}

// DeleteUnopenedTicket removes a row that never got a channel.
func (s *Store) DeleteUnopenedTicket(ctx context.Context, guildID string, id int64) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM tickets WHERE id = $1 AND guild_id = $2 AND channel_id = ''`, id, guildID)
	return err
}

func (s *Store) GetTicket(ctx context.Context, guildID string, id int64) (Ticket, error) {
	item, err := scanTicket(s.pool.QueryRow(ctx, `
		SELECT `+ticketColumns+` FROM tickets WHERE id = $1 AND guild_id = $2`, id, guildID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Ticket{}, ErrNotFound
	}
	return item, err
}

func (s *Store) OpenTicketFor(ctx context.Context, guildID, openerID string) (Ticket, error) {
	item, err := scanTicket(s.pool.QueryRow(ctx, `
		SELECT `+ticketColumns+`
		FROM tickets
		WHERE guild_id = $1 AND opener_id = $2 AND status = 'open'
		ORDER BY number DESC
		LIMIT 1`, guildID, openerID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Ticket{}, ErrNotFound
	}
	return item, err
}

func (s *Store) RetireMissingTicket(ctx context.Context, guildID string, id int64) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE tickets
		SET status = 'deleted', close_reason = 'The channel was already gone.', closed_at = COALESCE(closed_at, now())
		WHERE id = $1 AND guild_id = $2 AND status = 'open'`, id, guildID)
	return err
}

func (s *Store) ClaimTicket(ctx context.Context, guildID string, id int64, userID, name string) (Ticket, bool, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE tickets SET claimer_id = $3, claimer_name = $4
		WHERE id = $1 AND guild_id = $2 AND status = 'open' AND claimer_id = ''`,
		id, guildID, userID, name)
	if err != nil {
		return Ticket{}, false, err
	}
	item, err := s.GetTicket(ctx, guildID, id)
	if err != nil {
		return Ticket{}, false, err
	}
	return item, tag.RowsAffected() > 0, nil
}

func (s *Store) ReleaseTicket(ctx context.Context, guildID string, id int64) (Ticket, bool, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE tickets SET claimer_id = '', claimer_name = ''
		WHERE id = $1 AND guild_id = $2 AND status = 'open' AND claimer_id <> ''`, id, guildID)
	if err != nil {
		return Ticket{}, false, err
	}
	item, err := s.GetTicket(ctx, guildID, id)
	if err != nil {
		return Ticket{}, false, err
	}
	return item, tag.RowsAffected() > 0, nil
}

func (s *Store) CloseTicket(ctx context.Context, guildID string, id int64, byID, byName, reason, transcript string) (Ticket, bool, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE tickets SET
			status = 'closed',
			closed_by_id = $3,
			closed_by_name = $4,
			close_reason = $5,
			transcript = $6,
			closed_at = now()
		WHERE id = $1 AND guild_id = $2 AND status = 'open'`,
		id, guildID, byID, byName, reason, transcript)
	if err != nil {
		return Ticket{}, false, err
	}
	item, err := s.GetTicket(ctx, guildID, id)
	if err != nil {
		return Ticket{}, false, err
	}
	return item, tag.RowsAffected() > 0, nil
}

func (s *Store) ReopenTicket(ctx context.Context, guildID string, id int64) (Ticket, bool, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE tickets SET
			status = 'open',
			closed_by_id = '',
			closed_by_name = '',
			close_reason = '',
			transcript = '',
			closed_at = NULL
		WHERE id = $1 AND guild_id = $2 AND status = 'closed'`, id, guildID)
	if err != nil {
		return Ticket{}, false, err
	}
	item, err := s.GetTicket(ctx, guildID, id)
	if err != nil {
		return Ticket{}, false, err
	}
	return item, tag.RowsAffected() > 0, nil
}

func (s *Store) MarkTicketDeleted(ctx context.Context, guildID string, id int64) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE tickets SET status = 'deleted'
		WHERE id = $1 AND guild_id = $2 AND status = 'closed'`, id, guildID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *Store) ListTickets(ctx context.Context, guildID string) (open []Ticket, closed []Ticket, err error) {
	open = []Ticket{}
	closed = []Ticket{}
	open, err = s.queryTickets(ctx, `
		SELECT `+ticketColumns+`
		FROM tickets WHERE guild_id = $1 AND status = 'open'
		ORDER BY number`, guildID)
	if err != nil {
		return nil, nil, err
	}
	closed, err = s.queryTickets(ctx, `
		SELECT `+ticketColumns+`
		FROM tickets WHERE guild_id = $1 AND status <> 'open'
		ORDER BY closed_at DESC NULLS LAST, number DESC
		LIMIT 40`, guildID)
	if err != nil {
		return nil, nil, err
	}
	return open, closed, nil
}

func (s *Store) queryTickets(ctx context.Context, query, guildID string) ([]Ticket, error) {
	rows, err := s.pool.Query(ctx, query, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Ticket{}
	for rows.Next() {
		item, err := scanTicket(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
