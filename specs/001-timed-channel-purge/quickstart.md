# Quickstart: Timed Channel Purge

Prove the feature against the spec. Commands run from the repository root. Details live in [data-model.md](data-model.md), [contracts/purges-api.md](contracts/purges-api.md), and [contracts/dashboard.md](contracts/dashboard.md).

## Prerequisites

- Go toolchain pinned in `go.mod` (1.25)
- Node for the dashboard
- Docker, only for `make test-e2e` when Postgres is not already on port 5432

No new packages.

## 1. Domain rules

```sh
go test ./internal/domain -count=1 -run 'Purge|TooOld'
```

Expect pass. The tests cover:

- presets `1h`, `4h`, `12h`, `24h`, `7d` and their seconds
- custom 90 minutes, custom 60 minutes, and rejection of 10 minutes and 31 days
- rejection when both a preset and minutes are sent
- the age cutoff used by manual `/purge` and by a timed wipe

## 2. Unit race check

```sh
go test -race ./...
```

Expect pass. This includes the scheduler claim and the age split. It does not start Discord.

## 3. API and bot end-to-end

```sh
make test-e2e
```

Postgres and the Discord stand-in come up through the existing harness. Expect pass, including the new cases below.

API, as a dashboard admin:

1. `PUT /api/guilds/{guild}/purges/enabled` with `{ "enabled": false }` leaves any saved rows in place.
2. `POST` a `1h` preset on a text channel. The response `intervalSeconds` is 3600. With the feature off, `state` is `paused` and `nextRunAt` is null.
3. Enable the feature. `GET` shows `state` `scheduled` and `nextRunAt` about one hour ahead. The stand-in still has every seeded message.
4. `POST` `minutes: 90` on a second channel. The first channel is still `1h`.
5. `POST` `minutes: 10` and `minutes: 44640` returns 400 and the saved row is unchanged.
6. `POST` `minutes: 60` returns `preset` `""` and `intervalSeconds` 3600.
7. `DELETE` the second channel returns 204. `GET` still lists the first.
8. A category id or a voice channel id returns 400 `Choose a text channel.`

API, as the guest session: `PUT`, `POST`, and `DELETE` return 403.

Bot, using the harness to set `next_run_at` in the past and then calling the scheduler's due-purge function once per budget:

1. A channel with a member message, a bot message, and a pinned message, all recent, ends empty. No new message is posted. `GET /cases` does not gain a row. The schedule shows `removed` 3 and `skipped` 0.
2. A message older than 14 days stays. A recent message in the same channel is gone. `skipped` counts the old one.
3. More than 100 recent messages are all removed. This fails if the stand-in ignores `before`.
4. After the stand-in is taught to return 400 code 50034 for an old id, a wipe still leaves that old message.
5. Deleting the channel in the stand-in, then running the due purge, sets `state` to `missing` and a second run does not call Discord again.
6. Two due channels: the first finishes, the second is unchanged until a later call. Removing the second schedule before that call leaves its messages in place.
7. Manual `/purge` of 3 messages still replies `Deleted 3 messages.` and still skips messages older than the shared cutoff.

The harness move of `next_run_at` is test-only. The product API has no wipe-now route.

## 4. Dashboard

`make test-e2e` also runs `npm run e2e` in `web/`. Expect:

- The Moderation nav opens `Timed Channel Purge`.
- Turning the checkbox on and saving a 1 hour schedule shows the channel and does not claim the channel was wiped.

## Done when

- Domain, race, API, bot, and browser checks above pass.
- A server with the flag off keeps its messages across a due `next_run_at`.
- Enabling the flag or saving a schedule does not delete messages in that request.
