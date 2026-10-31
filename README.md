# SquirrelBot

SquirrelBot is a Discord moderation bot with a Go API and a SvelteKit admin dashboard. One process holds the Discord gateway, the HTTP API, and a scheduler. Postgres stores guild settings, cases, and sessions.

## What it does

Moderation follows the usual staff tools: warn, kick, ban, timeout, case history, purge, and slowmode. Automod can delete invites, repeated messages, mention piles, walls of caps, and blocked words. The bot also logs message and member events, posts a welcome message, assigns an auto role, and answers prefix commands. A prefix command can be a Lua script: it can reply, fill a Liquid string, react, delete the invoking message, and add or remove roles an admin allowed for that command.

Each server can give the bot its own display name and avatar. Members of that server see them in place of the application name and picture. The username stays the same on every server.

Reaction roles live in a channel the admin picks. SquirrelBot posts one message there. Clicking a reaction grants the role. Removing the reaction takes it away. The reaction roles page can create a new role or use one that already exists. The dashboard can create a logging, announcements, roles, or starboard channel under a name you choose. A logging channel is hidden from members.

Starboard reposts a message after enough reactions of the emoji an admin picks. The default emoji is ⭐ and the default limit is 3. A member's reaction on their own message does not count unless the server allows it. Images from NSFW channels stay out unless that is turned on. Channels on the ignore list are skipped, and the starboard channel itself is never posted.

Three jobs get extra attention:

- **Announcements.** The mod team posts an embed now, or schedules it. A role ping is optional. Saved templates use Liquid. `{{ server }}` and `{{ author }}` are filled in when a template is used, and other values are passed in at that time.
- **Admin Firecall Access.** This feature allows admins to grant temporary privilege escalation from one role to another for a given block of time.
- **Server application.** A new member must answer a questionnaire before the deadline. The first question is always "What is your age?". An answer under 18 removes them immediately. Finishing the questionnaire clears the deadline. They get a direct message, an optional warning, and a case if they are removed.

## Discord setup

1. Create an application and a bot at the Discord developer portal.
2. Turn on the **Server Members Intent** and the **Message Content Intent**.
3. Add this redirect: `{PUBLIC_URL}/auth/discord/callback`.
4. Sign in from the dashboard. Discord asks you to add the bot to a server where you have Manage Server. Sign in again to add another. The permission integer is `1099847265366` (view channels, send messages, embed links, read history, manage messages, manage channels, add reactions, change nickname, kick, ban, manage roles, and moderate members). Change Nickname lets the bot use a display name on that server. Add the server again if an older invite is missing it. The same install, without signing in, is:

```text
https://discord.com/oauth2/authorize?client_id=CLIENT_ID&permissions=1099847265366&integration_type=0&scope=bot%20applications.commands
```

Put the bot's role above every role it assigns. Escalation only grants roles on the allowlist.

Slash commands:

| Command | What it does |
| --- | --- |
| `/warn` `/kick` `/ban` `/timeout` `/untimeout` | Punish a member and record a case |
| `/cases` | Show the last cases for a member |
| `/purge` `/slowmode` | Clean a channel or set its slowmode |
| `/announce` | Post or schedule a mod team announcement |
| `/escalate` `/deescalate` `/escalations` | Lend an allowlisted role, end it, or list who has one |
| `/gate` | Show application status, or post the application message |

Prefix commands use the prefix from Welcome, default `!`. A saved command is a Lua 5.1 script. `reply` sends one message, and returning a string does the same when `reply` is not called. `liquid` fills `{{ author.name }}`, `{{ author.id }}`, `{{ author.display_name }}`, `{{ args }}`, `{{ guild.name }}`, and `{{ prefix }}`. `react` and `delete` act on the invoking message. `addRole` and `removeRole` only change roles selected for that command. Older commands that are plain text still send that text until you save them again.

## Run it locally

Copy `.env.example` to `.env` and fill in the Discord token, OAuth client id, OAuth secret, and a `SESSION_SECRET` of at least 32 characters. The process reads that file from the project directory. A variable already set in your shell wins over the file. `PUBLIC_URL` must be the origin you open in the browser. For dashboard development that is `http://localhost:5173`.

`make dev` starts Postgres, Air, and the dashboard at `http://localhost:5173`. Air rebuilds the bot when Go files, SQL migrations, or `.env` change. `make bot` and `make web` start those processes on their own, after `make db`.

`make build` writes `bin/squirrelbot` and `web/build`. `make image` builds the container `squirrelbot:0.1.0`. `make test` runs the Go unit tests. `make test-e2e` runs the API suite against Postgres and a Discord stand-in, then the browser suite against the dashboard. It starts the Compose Postgres if nothing is listening on port 5432.

`make up` runs the bot and Postgres in Docker. In that mode the dashboard is served by the bot on port 8080, so set `PUBLIC_URL` to `http://localhost:8080`. `make down` stops those containers.

`go run ./cmd/squirrelbot` starts one run and exits when you stop it.

## Kubernetes

The chart is `deploy/helm/squirrelbot`. It runs one replica with the Recreate strategy, a read-only root filesystem, and a non-root user. Liveness is `/healthz`. Readiness is `/readyz`, which stays down until Postgres answers and the Discord gateway is ready.

Supply secrets with `--set`, a private values file, or `existingSecret`. The secret keys are `DATABASE_URL`, `DISCORD_TOKEN`, `DISCORD_CLIENT_ID`, `DISCORD_CLIENT_SECRET`, and `SESSION_SECRET`.

```sh
helm upgrade --install squirrelbot deploy/helm/squirrelbot \
  --set publicUrl=https://mod.example.com \
  --set secrets.databaseUrl='postgres://squirrelbot:PASSWORD@db:5432/squirrelbot?sslmode=disable' \
  --set secrets.discordToken=TOKEN \
  --set secrets.discordClientId=CLIENT_ID \
  --set secrets.discordClientSecret=CLIENT_SECRET \
  --set secrets.sessionSecret=AT_LEAST_32_RANDOM_CHARACTERS
```

Set `postgres.enabled=true` and `postgres.auth.password` to start a bundled Postgres StatefulSet. That database is for a single cluster and a small server. Use an external Postgres for anything you need to restore.

`make image` builds `squirrelbot:0.1.0` from the repository root. Override the tag with `make image IMAGE=squirrelbot:dev`. Set `imagePullSecrets` when the registry is private.

## Releases

Pull requests run the tests. Merging a pull request into `main` tags that commit and publishes a GitHub release. The first release is `v0.1.0`. Later merged pull requests increment the patch number. Add `[skip release]` to the pull request title, the pull request body, or the commit message to skip publishing. A direct push to `main` does not publish a release.

CI pushes a tag. It does not push commits to `main`.

Each release includes:

- Archives for Linux and macOS, on amd64 and arm64. An archive contains `squirrelbot`, `web/build`, and `LICENSE`. Extract it and run `./squirrelbot` from that directory.
- The image `ghcr.io/boxingoctopuscreative/squirrelbot`, tagged with the version and with `latest`.
- The Helm chart, attached to the GitHub release and published to `oci://ghcr.io/boxingoctopuscreative/charts/squirrelbot`.

The published chart uses that GHCR image. The chart in this repository keeps the local image name `squirrelbot`, which matches `make image`. Install the published chart with the same `--set` values as the local chart, and set `--version` to the release you want.

CI sets the GHCR package visibility to match the repository.

## License

SquirrelBot is licensed under the Mozilla Public License 2.0. The terms are in [LICENSE](LICENSE).
