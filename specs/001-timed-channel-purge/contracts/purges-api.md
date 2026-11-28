# Contract: Timed purge HTTP API

All routes are under `/api/guilds/:guildID` and use the existing session, same-origin, and guild checks. A person without Manage Server or a dashboard access role receives 403 and `You need Manage Server, or a role chosen for dashboard access.`

Error bodies use the existing shape: `{ "error": "..." }`.

The guild object in `GET /api/guilds/:guildID` includes `purgeEnabled` (boolean).

## List schedules

`GET /api/guilds/:guildID/purges`

```json
{
  "enabled": true,
  "schedules": [
    {
      "channelId": "777777777777777782",
      "preset": "1h",
      "intervalSeconds": 3600,
      "state": "scheduled",
      "nextRunAt": "2026-11-27T23:19:00Z",
      "lastRunAt": null,
      "removed": 0,
      "skipped": 0,
      "error": ""
    }
  ]
}
```

- `preset` is `""` for a custom interval. `intervalSeconds` is always set.
- `state` is `paused`, `scheduled`, `running`, or `missing`. See `data-model.md`.
- `nextRunAt` is null when `state` is `paused` or `missing`. A paused row must not look due.
- `removed` and `skipped` are the last finished walk. They stay put while `state` is `running`.
- `error` is empty after a success. It is staff-facing text after a failure or a missing channel.
- Order is by channel id ascending so the list does not jump on each save.

## Turn the feature on or off

`PUT /api/guilds/:guildID/purges/enabled`

```json
{ "enabled": true }
```

Response:

```json
{ "enabled": true }
```

- The body must include `enabled`. Otherwise 400 `Choose on or off.`
- Turning it on rewrites the next run for waiting schedules to one interval after this request. It does not delete messages.
- Turning it off does not delete schedules.

## Save a schedule

`POST /api/guilds/:guildID/purges`

Preset:

```json
{ "channelId": "777777777777777782", "preset": "4h" }
```

Custom:

```json
{ "channelId": "777777777777777782", "minutes": 90 }
```

Response: one schedule object, the same fields as a list item. `state` follows the current toggle. `nextRunAt` is one interval after this save when the feature is on, and null when it is off.

Rules:

- Exactly one of `preset` and `minutes`. Otherwise 400 `Choose a common interval or a custom length.`
- `preset` is `1h`, `4h`, `12h`, `24h`, or `7d`.
- `minutes` is an integer from 15 through 43200. Otherwise 400 `Custom intervals are from 15 minutes to 30 days.` A value of 60 is accepted.
- `channelId` must be a text or announcement channel the bot can see. Otherwise 400 `Choose a text channel.`
- The bot must be connected. Otherwise 503 with the existing "still connecting" error.
- A second save for the same channel replaces the interval and restarts the wait. Other channels are unchanged. Messages are not deleted by this request.
- The previous last-run counts remain until the next walk finishes.

## Remove a schedule

`DELETE /api/guilds/:guildID/purges/:channelID`

- 204 when the row was deleted.
- 404 `That channel has no schedule.` when it was already gone.
- Other channels are unchanged. No messages are deleted by this request.

## What this API does not do

- It does not wipe a channel at save time or when the toggle is turned on.
- It does not create a moderation case.
- It does not change `/purge`.
