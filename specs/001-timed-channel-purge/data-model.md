# Data Model: Timed Channel Purge

## Guild flag

Stored on the existing guild row.

| Field | Type | Rules |
| --- | --- | --- |
| `purge_enabled` | boolean, not null, default `false` | Server-wide on/off. Off keeps schedules and blocks new claims. |

JSON name: `purgeEnabled`.

Turning the flag from off to on sets `next_run_at = now() + interval_seconds` on every `scheduled` row for that guild. It does not change a `running` row or a `missing` row. Turning the flag off does not rewrite schedules.

## Channel schedule

Table `channel_purges`. Primary key `(guild_id, channel_id)`. `guild_id` references `guilds(id)` and deletes with the guild.

| Field | Type | Rules |
| --- | --- | --- |
| `guild_id` | text | Existing guild id. |
| `channel_id` | text | Discord snowflake. One row per channel. |
| `preset` | text, not null, default `''` | `''`, `1h`, `4h`, `12h`, `24h`, or `7d`. Empty means custom. |
| `interval_seconds` | integer, not null | 900 through 2,592,000. Must match the preset when the preset is set. |
| `next_run_at` | timestamptz, null | When the next claim may happen. Null only while `missing`. |
| `status` | text, not null | `scheduled`, `running`, or `missing`. |
| `generation` | bigint, not null, default 1 | Incremented on every save and on delete's replacement. A tick stops when the loaded generation differs. |
| `cursor_id` | text, not null, default `''` | Oldest message id already handled in the current walk. Empty means start at the newest. |
| `removed_count` | integer, not null, default 0 | In-progress removed total for the current walk. |
| `skipped_count` | integer, not null, default 0 | In-progress skipped total for the current walk. |
| `last_run_at` | timestamptz, null | When the last walk finished or failed. |
| `last_removed` | integer, not null, default 0 | Removed count from that walk. |
| `last_skipped` | integer, not null, default 0 | Skipped count from that walk. |
| `error` | text, not null, default `''` | Empty after success. Staff-facing text after a failure or a missing channel. |
| `updated_at` | timestamptz, not null | Set on every write. |

Index for the claim: `(next_run_at)` where `status = 'scheduled'`.

A save replaces the row for that channel: new preset and interval, `status = scheduled`, `generation` incremented, cursor cleared, in-progress counts cleared, `next_run_at` one interval from the save. The previous `last_*` values stay so the panel still shows the last finished wipe. The save does not delete messages.

A remove deletes that row only.

## Domain values

Pure functions in `internal/domain`. No database and no Discord imports.

| Name | Values |
| --- | --- |
| Presets | `1h` 3600, `4h` 14400, `12h` 43200, `24h` 86400, `7d` 604800 |
| Custom minutes | Integers 15 through 43200 |
| Age cutoff | 13 days, shared with manual `/purge` |

`TooOldToBulkDelete(sent, now)` is true when `sent` is before `now` minus 13 days. Manual purge calls this instead of inlining the duration. The staff text stays "older than 14 days."

Validation rejects:

- both a preset and custom minutes, or neither
- an unknown preset
- minutes outside 15 through 43200
- a preset whose seconds were altered
- a channel id that is not a snowflake
- a channel whose type is not guild text or guild announcement

## State transitions

```text
scheduled --claim (toggle on, next_run_at <= now)--> running
running   --pages remain--> running (cursor and in-progress counts saved)
running   --walk finished--> scheduled (last_* copied, error cleared, cursor cleared, next_run_at = now + interval)
running   --403 or other page error--> scheduled (last_* copied, error set, next_run_at = now + interval)
running   --404 code 10003--> missing (error set, next_run_at null, cursor cleared)
running   --save--> scheduled (generation bumped, cursor cleared, next_run_at reset, tick stops)
running   --delete--> row gone (tick stops)
missing   --save of a channel that exists again--> scheduled
scheduled --delete--> row gone

startup: running --> scheduled, cursor cleared, next_run_at = now()
```

API `state` is not the stored status:

| Condition | `state` |
| --- | --- |
| Toggle off and status is `scheduled` | `paused` |
| Toggle on and status is `scheduled` | `scheduled` |
| Status is `running` | `running` |
| Status is `missing` | `missing` |

A running wipe stays `running` in the API after the toggle is turned off, until that walk finishes. The claim will not start another one.

## Relationships

- A guild has one toggle and zero or more channel schedules.
- A channel schedule has at most one in-progress walk, stored on the same row.
- A wipe result is the `last_*` fields and `error` on that row. It is not a moderation case.
- Schedules do not reference threads, forum posts, or the manual purge command.

## Migration

`internal/db/migrations/019_channel_purges.sql` adds `guilds.purge_enabled` and creates `channel_purges`. Existing servers get the flag off and no schedules.
