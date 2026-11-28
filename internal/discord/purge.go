package discord

import (
	"context"
	"errors"
	"time"

	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/BoxingOctopus/squirrelbot/internal/store"
	"github.com/bwmarrin/discordgo"
)

const purgePageBudget = 10

// RunDuePurges continues one channel wipe, or starts the oldest due wipe.
// The scheduler calls this. Tests call it directly.
func (b *Bot) RunDuePurges(ctx context.Context) {
	row, ok, err := b.store.RunningPurge(ctx)
	if err != nil {
		b.log.Error("read running purge", "err", err)
		return
	}
	if !ok {
		row, ok, err = b.store.ClaimDuePurge(ctx)
		if err != nil {
			b.log.Error("claim purge", "err", err)
			return
		}
		if !ok {
			return
		}
	}
	b.advancePurge(ctx, row)
}

func (b *Bot) advancePurge(ctx context.Context, row store.ChannelPurge) {
	cursor := row.CursorID
	removed := row.RemovedCount
	skipped := row.SkippedCount
	for page := 0; page < purgePageBudget; page++ {
		if !b.purgeStillRunning(ctx, row) {
			return
		}
		before := cursor
		messages, err := b.session.ChannelMessages(row.ChannelID, 100, before, "", "")
		if err != nil {
			b.stopPurge(ctx, row, removed, skipped, err)
			return
		}
		if len(messages) == 0 {
			if err := b.store.FinishPurge(ctx, row.GuildID, row.ChannelID, row.Generation, removed, skipped); err != nil {
				b.log.Error("finish purge", "guild", row.GuildID, "channel", row.ChannelID, "err", err)
			}
			return
		}
		ids := make([]string, 0, len(messages))
		pageSkipped := 0
		now := time.Now()
		for _, message := range messages {
			if domain.TooOldToBulkDelete(message.Timestamp, now) {
				pageSkipped++
				continue
			}
			ids = append(ids, message.ID)
			b.muted.mark(message.ID)
		}
		if err := deleteMessageIDs(b.session, row.ChannelID, ids); err != nil {
			b.stopPurge(ctx, row, removed, skipped, err)
			return
		}
		removed += len(ids)
		skipped += pageSkipped
		cursor = messages[len(messages)-1].ID
		saved, err := b.store.SavePurgeProgress(ctx, row.GuildID, row.ChannelID, row.Generation, cursor, removed, skipped)
		if err != nil {
			b.log.Error("save purge progress", "guild", row.GuildID, "channel", row.ChannelID, "err", err)
			return
		}
		if !saved {
			return
		}
		if len(messages) < 100 {
			if err := b.store.FinishPurge(ctx, row.GuildID, row.ChannelID, row.Generation, removed, skipped); err != nil {
				b.log.Error("finish purge", "guild", row.GuildID, "channel", row.ChannelID, "err", err)
			}
			return
		}
	}
}

func (b *Bot) purgeStillRunning(ctx context.Context, row store.ChannelPurge) bool {
	current, err := b.store.GetChannelPurge(ctx, row.GuildID, row.ChannelID)
	if err != nil || current.Generation != row.Generation || current.Status != "running" {
		return false
	}
	return true
}

func (b *Bot) stopPurge(ctx context.Context, row store.ChannelPurge, removed, skipped int, cause error) {
	if !b.purgeStillRunning(ctx, row) {
		return
	}
	missing, text := purgeFailure(cause)
	var err error
	if missing {
		err = b.store.MissPurge(ctx, row.GuildID, row.ChannelID, row.Generation, removed, skipped)
	} else {
		err = b.store.FailPurge(ctx, row.GuildID, row.ChannelID, row.Generation, removed, skipped, text)
	}
	if err != nil {
		b.log.Error("record purge failure", "guild", row.GuildID, "channel", row.ChannelID, "err", err)
		return
	}
	b.log.Warn("channel purge", "guild", row.GuildID, "channel", row.ChannelID, "err", cause)
}

func purgeFailure(err error) (missing bool, text string) {
	if statusOf(err) == 404 && discordCode(err) == 10003 {
		return true, "That channel is gone."
	}
	if statusOf(err) == 403 {
		return false, "Discord refused that. Check my role position and permissions."
	}
	return false, "Discord refused that action."
}

func discordCode(err error) int {
	var rest *discordgo.RESTError
	if errors.As(err, &rest) && rest.Message != nil {
		return rest.Message.Code
	}
	return 0
}

func deleteMessageIDs(session *discordgo.Session, channelID string, ids []string) error {
	for len(ids) > 0 {
		n := len(ids)
		if n > 100 {
			n = 100
		}
		chunk := ids[:n]
		ids = ids[n:]
		var err error
		if len(chunk) == 1 {
			err = session.ChannelMessageDelete(channelID, chunk[0])
		} else {
			err = session.ChannelMessagesBulkDelete(channelID, chunk)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
