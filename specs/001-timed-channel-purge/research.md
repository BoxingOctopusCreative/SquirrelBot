# Research: Timed Channel Purge

## Server-wide toggle

- **Decision**: Add `guilds.purge_enabled`, default `false`. The admin panel saves it with `PUT /api/guilds/:guildID/purges/enabled`. Turning it on rewrites `next_run_at` for every waiting schedule to one full interval after that save. Turning it off only flips the flag.
- **Rationale**: Levels default off. Welcome and announcements default on because those columns arrived after the behavior already existed. A new wipe must not start deleting on servers that have not opted in. Resetting `next_run_at` on enable is what the spec requires, so this toggle cannot use the generic flag saver alone.
- **Alternatives considered**: Default on, matching the migration that added the older toggles. Rejected because those defaults preserved existing behavior. Reuse `putGuildFlag` without touching schedules. Rejected because enable would leave a past `next_run_at` and the next tick would wipe immediately.

## Where a schedule lives

- **Decision**: One Postgres row per guild and channel in `channel_purges`. The row stores a preset key or an empty preset plus `interval_seconds`, the next run, a status, a page cursor, and the last result. The scheduler in `internal/discord/scheduler.go` claims due rows. No second loop, no second process.
- **Rationale**: Announcements already claim due work with `UPDATE ... FOR UPDATE SKIP LOCKED` inside the 15 second scheduler, and `RecoverStuckWork` runs before the gateway opens. A channel wipe is the same shape: durable, recoverable, and single-threaded at the scheduler.
- **Alternatives considered**: A column array of channel ids on `guilds`. Rejected because each channel has its own interval, cursor, and result. An in-memory timer. Rejected because a restart would drop due work (Principle I).

## Preset versus custom

- **Decision**: Presets are `1h` (3600s), `4h` (14400s), `12h` (43200s), `24h` (86400s), and `7d` (604800s). Custom is whole minutes from 15 through 43200, stored as seconds with an empty preset. Sixty minutes is a valid custom value even though `1h` is the same length. Domain code checks the pair. The next run uses `interval_seconds` only.
- **Rationale**: The spec asks to tell a 60 minute custom save apart from the 1 hour preset. Duration alone cannot do that. The closed list matches the spec. The bounds match 15 minutes and 30 days.
- **Alternatives considered**: Store only seconds and infer the preset. Rejected because 3600 seconds is both `1h` and 60 custom minutes. A clock time such as 09:00. Rejected in the spec.

## What gets deleted

- **Decision**: Walk the channel newest-first with `GET /channels/{id}/messages?limit=100&before=`. Split each page with the same age rule as `/purge`: a message older than 13 days is kept and counted as skipped. Younger messages are deleted, including pinned messages and bot messages. Two or more ids go to bulk delete, in chunks of at most 100. One id uses a single delete. The walk continues through older pages so the skipped count covers the whole channel. It does not call the thread list, and it does not post in the channel or open a case.
- **Rationale**: Discord error 50034 rejects a bulk delete that includes any message older than 14 days. The manual purge already avoids that with a 13 day cutoff and tells staff the messages are older than 14 days. Sharing that rule keeps `/purge` behavior and the timed wipe on one decision (Principle II). Bulk delete does remove pins; tools skip pins only when they want to keep them. This spec wants pins gone. Parent-channel history does not include thread or forum-post messages, so those stay without a special case.
- **Alternatives considered**: A strict 14 day cutoff. Rejected because a message near the edge returns 50034 and can fail the whole chunk. Deleting old messages one by one. Rejected by the spec. Skipping pins. Rejected by the spec. Cloning the channel to drop history older than 14 days. Rejected because it changes the channel id and everything that points at it.

## How a long wipe shares the scheduler

- **Decision**: One wipe is active at a time. Each 15 second tick continues it for at most 10 message pages, saves the cursor and the in-progress counts, then returns to announcements, voice XP, firecall expiry, and the application gate. A channel with 1,000 eligible messages finishes in that one budget. Older history continues on later ticks. The panel keeps the previous finished result until the walk ends.
- **Rationale**: The scheduler is one function on one process. A full history walk on that same call would delay kicks and announcements for minutes. Ten pages covers the success target of 1,000 eligible messages without a second ticker. Ten pages of 100 is also one bulk-delete chunk each, which matches Discord's 100 id cap.
- **Alternatives considered**: Run each wipe to completion inline. Rejected because a large history blocks the rest of the scheduler. A pool of goroutines. Rejected because the ticker can pace the work itself, and one active wipe is easier to recover after a crash. A second scheduler process. Rejected by Principle I.

## Failures, overlap, and a restart

- **Decision**: Claim sets `status` to `running`. The claim ignores a guild whose toggle is off and ignores a row that is already running or missing. A 404 with Discord code 10003 marks the row `missing` and stops future claims. A 403 keeps the schedule, stores the error, and sets the next run one interval out. Any other page error does the same with the counts gathered so far. Saving or deleting the row bumps a generation; the tick reloads the row and stops if the generation changed. Startup recovery sets `running` back to `scheduled`, clears the cursor, and sets `next_run_at` to now so the channel runs once.
- **Rationale**: One row per channel is the overlap lock (FR-015). Recovery matches announcements and firecall grants, and it collapses missed intervals into one run (FR-016). A generation stops a tick from writing over an admin's newer save. Missing channels must not retry forever (FR-019).
- **Alternatives considered**: Delete the row on 404. Rejected because the panel has to show that the channel is gone. Retry a 403 on the next tick. Rejected because the spec waits for the next interval. Leave `running` across a restart. Rejected because nothing would be left to continue the cursor, and the row would never be claimed again.

## Who can edit, and which channels

- **Decision**: The new routes sit on the existing `/api/guilds/:guildID` group. `requireGuild` already allows Manage Server or a dashboard access role. The channel must be a guild text channel or a guild announcement channel, the same two types the dashboard channel list already returns. Voice, categories, forums, and threads are rejected. The bot's Discord permission is checked by the API at runtime: missing Manage Messages becomes the 403 path above.
- **Rationale**: The spec gives this feature to the same people who change other settings. No new slash command. Announcement channels hold messages and already appear in the channel picker. Threads are separate channels and are not in that picker.
- **Alternatives considered**: A moderator slash command to set the schedule. Rejected in the spec. Accept any snowflake and let the wipe fail later. Rejected because a voice channel id should fail at save time.

## Tests and the Discord stand-in

- **Decision**: Domain tests cover presets, bounds, the 60 minute custom case, and the shared age cutoff. API end-to-end tests cover auth, the toggle, validation, replace, and remove. Bot end-to-end tests call the same due-purge function the scheduler calls, after the harness moves `next_run_at` into the past. The stand-in must honor `before`, return 404 code 10003 for an unknown channel, and return 400 code 50034 when a bulk delete includes a message older than 14 days. The browser suite gains the new nav entry and one save flow.
- **Rationale**: The scheduler does not run in the end-to-end stack today. Tests call bot methods directly. The stand-in currently returns the newest page only and deletes whatever ids it is given, so a wipe of more than 100 messages, or a wipe that forgot the age filter, would look successful for the wrong reason. There is no product "wipe now" route.
- **Alternatives considered**: Wait 15 real minutes in the test. Rejected. Add a public wipe-now endpoint. Rejected by the spec. Leave the stand-in unchanged and only unit-test the page split. Rejected because the delete call is the behavior staff see.
