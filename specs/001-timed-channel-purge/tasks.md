# Tasks: Timed Channel Purge

**Input**: Design documents from `/specs/001-timed-channel-purge/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Included. The plan and quickstart require domain tests, API tests, bot tests, and a dashboard check.

**Organization**: Tasks are grouped by user story so each story can be tested on its own.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)

## Phase 1: Setup

**Purpose**: Confirm the existing module layout. No new dependencies.

- [X] T001 Confirm Go 1.25, Fiber, pgx, and discordgo stay the stack in `go.mod`

## Phase 2: Foundational

**Purpose**: Shared rules, storage, and the scheduler hook that every story uses.

- [X] T002 Add `TooOldToBulkDelete` and `PurgeChannelAllowed` in `internal/domain/purge.go`, with tests in `internal/domain/purge_test.go`. Cite `data-model.md` `PurgeMaxAge` and channel types 0 and 5
- [X] T003 Point manual `/purge` at `TooOldToBulkDelete` in `internal/discord/moderation.go`
- [X] T004 Add `guilds.purge_enabled` (`data-model.md` Guild flag, default false) and `channel_purges` in `internal/db/migrations/019_channel_purges.sql`. Cite `data-model.md` `Channel schedule.preset`, `interval_seconds`, `status`, `generation`, and count checks
- [X] T005 Add purge store methods and the running-row reset in `internal/store/purge.go`, `internal/store/guild.go`, and `internal/store/settings.go`
- [X] T006 Teach the Discord stand-in `before`, unknown channel 10003, forbidden 50013, and bulk-delete 50034 in `internal/discordmock/channel.go` and `internal/discordmock/server.go`
- [X] T007 Call `RunDuePurges` from the existing tick in `internal/discord/scheduler.go`

**Checkpoint**: A due row can be claimed, and a restart clears a `running` row.

## Phase 3: User Story 1 - Schedule a channel wipe on a common interval (Priority: P1)

**Goal**: An admin turns the feature on, saves 1 hour, 4 hours, 12 hours, 24 hours, or 7 days, and the channel is wiped when the interval elapses.

**Independent Test**: Save a 1 hour preset, move `next_run_at` into the past, run one scheduler pass, and see recent messages gone.

### Tests for User Story 1

- [X] T008 [P] [US1] Cover preset saves, the paused state, and the enable wait in `e2e/api_test.go`
- [X] T009 [P] [US1] Cover a due wipe of member, bot, and pinned messages in `e2e/bot_test.go`

### Implementation for User Story 1

- [X] T010 [US1] Parse preset keys in `internal/domain/purge.go`. Cite `data-model.md` `PurgeInterval.Preset` and `PurgeInterval.Seconds`
- [X] T011 [US1] Walk and delete pages in `internal/discord/purge.go`
- [X] T012 [US1] Add list, enable, and save routes in `internal/api/server.go` and `internal/api/purge.go` per `contracts/purges-api.md`
- [X] T013 [US1] Add the Timed Channel Purge page in `web/src/routes/app/[guildId]/purges/+page.svelte`, `web/src/routes/app/[guildId]/purges/+page.ts`, `web/src/routes/app/[guildId]/+layout.svelte`, and `web/src/lib/types.ts` per `contracts/dashboard.md`

**Checkpoint**: A preset schedule wipes a channel and the panel shows the result.

## Phase 4: User Story 2 - Set a custom interval (Priority: P2)

**Goal**: An admin saves a minute count from 15 through 43200 without changing other channels.

**Independent Test**: Save 90 minutes on one channel and 60 minutes on another. Reject 10 minutes and 31 days.

### Tests for User Story 2

- [X] T014 [US2] Cover custom minutes, rejection, and an unchanged sibling schedule in `e2e/api_test.go`

### Implementation for User Story 2

- [X] T015 [US2] Accept custom minutes in `internal/domain/purge.go`. Cite `data-model.md` `PurgeInterval.Seconds` bounds of 900 through 2592000, and an empty `PurgeInterval.Preset`
- [X] T016 [US2] Show the custom minute field on `web/src/routes/app/[guildId]/purges/+page.svelte`

**Checkpoint**: Custom and preset schedules can sit on different channels.

## Phase 5: User Story 3 - Stop wipes and read what happened (Priority: P3)

**Goal**: Staff can pause every wipe, remove one schedule, and see skips, permission failures, and missing channels.

**Independent Test**: Schedule two channels, wipe one, remove the other, and confirm the removed channel keeps its messages. A paused feature wipes nothing.

### Tests for User Story 3

- [X] T017 [US3] Cover delete, guest rejection, old messages, a wide channel, a forbidden channel, a missing channel, and a paused feature in `e2e/api_test.go` and `e2e/bot_test.go`

### Implementation for User Story 3

- [X] T018 [US3] Record skip, failure, and missing results in `internal/discord/purge.go` and `internal/store/purge.go`. Cite `data-model.md` `Channel schedule.status` and `Channel schedule.error`
- [X] T019 [US3] Show paused, in progress, and gone states on `web/src/routes/app/[guildId]/purges/+page.svelte`
- [X] T020 [US3] Let tests move a row due without a product wipe-now route in `e2e/harness/stack.go`

**Checkpoint**: Stopping a schedule or the feature leaves the other channels alone.

## Phase 6: Polish

**Purpose**: Dashboard navigation coverage from `quickstart.md`

- [X] T021 Add Timed Channel Purge to the section walk and the save flow in `web/e2e/dashboard.spec.ts`

## Dependencies & Execution Order

### Phase Dependencies

- Setup has no dependencies.
- Foundational blocks every story.
- User Story 1 can start after Foundational. User Stories 2 and 3 build on that save and wipe path.
- Polish runs after the page exists.

### User Story Dependencies

- **User Story 1 (P1)**: First wipe and preset save. No dependency on US2 or US3.
- **User Story 2 (P2)**: Uses the US1 save route. Independently testable with a minute count.
- **User Story 3 (P3)**: Uses the US1 wipe. Independently testable by pausing, removing, and failing.

### Parallel Opportunities

- T008 and T009 can be written together.
- T002 and T006 touch different files and can proceed together after T001.

### Parallel Example: User Story 1

```bash
# API contract and bot wipe can be specified together:
Task: "Cover preset saves in e2e/api_test.go"
Task: "Cover a due wipe in e2e/bot_test.go"
```

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Finish Phase 1 and Phase 2.
2. Finish User Story 1.
3. Stop and run the 1 hour wipe test.

### Incremental Delivery

1. Preset wipe.
2. Custom minutes.
3. Pause, remove, and failure states.
4. Dashboard navigation check.

## Notes

- One active wipe, ten pages per scheduler tick.
- No moderation case and no message posted in the channel.
- The manual purge command keeps its reply and its age cutoff.
