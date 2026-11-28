# Implementation Plan: Timed Channel Purge

**Branch**: `001-timed-channel-purge` | **Date**: 2026-11-27 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/001-timed-channel-purge/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command; its definition describes the execution workflow.

## Summary

Staff turn Timed Channel Purge on in the admin panel and attach one interval to each text channel. The interval is a preset (1 hour, 4 hours, 12 hours, 24 hours, 7 days) or a custom length from 15 minutes through 30 days. The existing 15 second scheduler wipes due channels, up to 10 message pages per tick, and records how many messages were removed or skipped. Messages older than the manual purge cutoff stay. Pins and bot messages go. Nothing is posted in the channel, and no moderation case is opened.

The design follows [research.md](research.md). The flag defaults off. Enabling it, or saving a schedule, restarts the wait and does not delete messages.

## Technical Context

**Language/Version**: Go 1.25 (module in `go.mod`) and the SvelteKit dashboard (Svelte 5)

**Primary Dependencies**: Fiber, pgx, discordgo, the existing Discord stand-in. No new packages.

**Storage**: Postgres. New migration `internal/db/migrations/019_channel_purges.sql`. Guild flag plus one row per scheduled channel.

**Testing**: `go test` for domain rules, `go test -race ./...`, `make test-e2e` for the API, the bot against the stand-in, and the Playwright dashboard suite

**Target Platform**: The single squirrelbot process (gateway, HTTP API, and scheduler) on Linux, with the static dashboard

**Performance Goals**: A due channel's eligible messages are gone within 5 minutes. 1,000 eligible messages finish inside one 10-page tick, inside the 10 minute spec target. The tick then returns to the rest of the scheduler.

**Constraints**: One process and one scheduler (Principle I). Age rule and interval checks live in `internal/domain` (Principle II). No moderation case (Principle III). Same dashboard access as other settings (Principle IV). Discord bulk delete rejects any id older than 14 days (error 50034) and accepts at most 100 ids. The manual purge's 13 day cutoff is the shared rule. Message text is not stored or logged.

**Scale/Scope**: One schedule per text or announcement channel. One active wipe at a time. Other due channels wait for the next ticks. History older than the cutoff is counted across later ticks, 10 pages at a time.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Before research

| Gate | Result |
| --- | --- |
| I. One Process | Pass. The wipe is more scheduled work for the scheduler that already runs in this process. Postgres remains the record. No second replica. |
| II. Rules Before Effects | Pass. Presets, custom bounds, and the age cutoff are pure decisions with tests. Handlers apply them afterward. |
| III. A Case for Every Punishment | Pass. A channel wipe is not a warn, kick, ban, timeout, or application removal. The spec forbids a case. Manual punishments stay as they are. |
| IV. Least Privilege | Pass. Edits use the existing guild session check. No new command, secret, or script capability. |
| V. Proven Before Merge | Pass. The plan adds domain tests, API tests, bot tests, and a dashboard test to the suites CI already runs. |

No gate failed. Phase 0 proceeded.

### After design

| Gate | Result |
| --- | --- |
| I. One Process | Pass. `RunScheduler` calls the wipe on the same tick. A 10-page budget returns the tick to announcements, voice XP, firecall expiry, and the application gate. `RecoverStuckWork` resets a `running` row before the gateway opens. |
| II. Rules Before Effects | Pass. `internal/domain` owns presets, minute bounds, and `TooOldToBulkDelete`. Store code is SQL only. Discord code lists, deletes, and writes the result. Manual `/purge` calls the same age function, with the same 13 day window it has now. |
| III. A Case for Every Punishment | Pass. The wipe path does not call `CreateCase`. The bot test asserts `/cases` is unchanged. |
| IV. Least Privilege | Pass. Routes stay behind `requireGuild`. Logs record guild id, channel id, counts, and the error. They do not record message text. |
| V. Proven Before Merge | Pass. [quickstart.md](quickstart.md) maps each spec story onto `go test`, `make test-e2e`, and the existing browser suite. |

No gate failed after design. No exception is required.

## Project Structure

### Documentation (this feature)

```text
specs/001-timed-channel-purge/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
│   ├── purges-api.md
│   └── dashboard.md
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
internal/domain/purge.go                 # presets, bounds, age cutoff
internal/domain/purge_test.go
internal/discord/moderation.go           # /purge calls the shared age cutoff
internal/discord/purge.go                # list, delete, record the result
internal/discord/scheduler.go            # one call from the existing tick
internal/store/purge.go                  # CRUD, claim, recovery update
internal/store/settings.go               # RecoverStuckWork gains the running-row reset
internal/store/guild.go                  # purgeEnabled on the guild row
internal/db/migrations/019_channel_purges.sql
internal/api/server.go                   # routes
internal/api/handlers.go                 # list, toggle, save, remove
internal/discordmock/channel.go          # before, unknown channel, error 50034
e2e/harness/stack.go                     # test-only way to mark a row due
e2e/api_test.go
e2e/bot_test.go
web/src/lib/types.ts                     # purgeEnabled and schedule types
web/src/routes/app/[guildId]/+layout.svelte
web/src/routes/app/[guildId]/purges/+page.svelte
web/e2e/dashboard.spec.ts
```

**Structure Decision**: This repo is one Go process plus a SvelteKit dashboard. The feature adds a domain file, a store file, a Discord file, one migration, three HTTP routes, and one dashboard page. It does not add a service, a queue, or a package.

## Complexity Tracking

No constitution violations need a justification.

## Phase 0 and Phase 1

Phase 0 decisions are in [research.md](research.md). Phase 1 artifacts:

- [data-model.md](data-model.md) for the flag, the schedule row, and the states
- [contracts/purges-api.md](contracts/purges-api.md) for the HTTP API
- [contracts/dashboard.md](contracts/dashboard.md) for the page
- [quickstart.md](quickstart.md) for the checks that prove the spec

## Wipe behavior the tasks must keep

- Claim one `scheduled` row whose `next_run_at` is due, and only when `purge_enabled` is true. Set it to `running`.
- If a row is already `running`, continue that row instead of claiming another.
- Each tick handles at most 10 pages of 100 messages, newest first, using `before`.
- Skip ids the shared age rule rejects. Bulk-delete the rest in chunks of 2 to 100. A single id uses one delete. Include pinned and bot messages.
- On a finished walk, copy the counts to `last_removed` and `last_skipped`, clear the error and the cursor, and set `next_run_at` one interval ahead.
- On 404 code 10003, mark `missing` and do not claim it again.
- On 403 or a page error, store the error, keep the schedule, and wait one interval.
- Reload `generation` between pages. A save or a delete stops the tick without writing over the new row.
- Do not send a channel message. Do not create a case.
