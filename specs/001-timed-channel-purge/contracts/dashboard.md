# Contract: Timed Channel Purge page

Route: `/app/:guildId/purges`

Navigation label, page heading, and document title: `Timed Channel Purge`. The item sits in the Moderation group, after Automod.

The page reads `purgeEnabled` from the guild bundle and the schedule list from `GET /api/guilds/:guildID/purges`.

## Toggle

- Checkbox label: `Timed purge on`.
- Save posts `PUT /api/guilds/:guildID/purges/enabled`.
- Notice on success: `Saved. Timed purge updated.`
- While the box is unchecked, the page says schedules stay saved and will not run.
- While unchecked, each schedule shows `Paused` and does not show a due time.

## Schedule form

- Channel control is the existing `ChannelSelect`. It takes the value from the change event. It does not use `bind:value`.
- Interval control is a preset list: 1 hour, 4 hours, 12 hours, 24 hours, 7 days, and Custom.
- Custom shows a minute field. The field accepts 15 through 43200.
- Save posts `POST /api/guilds/:guildID/purges` with either `preset` or `minutes`.
- The form does not offer a wipe-now action.

## Schedule list

Each row shows:

- the channel name when the bundle still has that channel, otherwise the channel id
- the interval label (`1 hour`, or `90 minutes` for a custom value)
- the next run, or `Paused`, or nothing when the channel is missing
- the last run time, removed count, and skipped count
- the error text when it is non-empty
- `That channel is gone.` when `state` is `missing`
- an Edit button that opens an interval form inside that row. The channel and interval fields above stay as they are. Update posts the same `POST`. Cancel closes the row form
- a Remove button that calls `DELETE` for that channel only

A running wipe shows `Wipe in progress` and keeps the previous removed and skipped counts on screen.

## Browser coverage

`web/e2e/dashboard.spec.ts` already opens every nav item by name. Add `Timed Channel Purge` to that list. Add one flow that turns the feature on, saves a 1 hour schedule on a text channel, and sees that channel in the list without a success message that says messages were deleted.
