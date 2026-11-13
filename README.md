# SquirrelBot

SquirrelBot is a Discord moderation bot with a Go API and a SvelteKit admin dashboard. One process holds the Discord gateway, the HTTP API, and a scheduler. Postgres stores guild settings, cases, and sessions.

## What it does

Moderation follows the usual staff tools: warn, kick, ban, timeout, case history, purge, and slowmode. Automod can delete invites, repeated messages, mention piles, walls of caps, and blocked words. The bot also logs message and member events, posts a welcome message, assigns an auto role, and answers prefix commands. A prefix command can be a Lua script: it can reply, fill a Liquid string, react, delete the invoking message, and add or remove roles an admin allowed for that command.

Each server can give the bot its own display name and avatar. Members of that server see them in place of the application name and picture. The username stays the same on every server.

Reaction roles live in a channel the admin picks. SquirrelBot posts one message there. Clicking a reaction grants the role. Removing the reaction takes it away. The reaction roles page can create a new role or use one that already exists. The dashboard can create a logging, announcements, roles, or starboard channel under a name you choose. A logging channel is hidden from members.

Starboard reposts a message after enough reactions of the emoji an admin picks. The default emoji is ⭐ and the default limit is 3. A member's reaction on their own message does not count unless the server allows it. Images from NSFW channels stay out unless that is turned on. Channels on the ignore list are skipped, and the starboard channel itself is never posted.

Anonymous confessions start with `/confession`, or with the Submit a Confession button on an approved post. That button opens a form for the text and an optional file. The note goes to a review channel, tags the mod role, and shows Approve and Reject. An approved note is posted in the confessions channel without the author's name, with the same button for the next person. A review channel created from the dashboard is hidden from members. Moderators choose either channel in the dashboard, or with `/confession-review` and `/confession-channel`.

Three jobs get extra attention:

- **Announcements.** The mod team posts an embed now, or schedules it. A role ping is optional. Saved templates use Liquid. `{{ server }}` and `{{ author }}` are filled in when a template is used, and other values are passed in at that time.
- **Admin Firecall Access.** This feature allows admins to grant temporary privilege escalation from one role to another for a given block of time.
- **Server application.** A new member must answer a questionnaire before the deadline. The first question is always "What is your age?". An answer under 18 removes them immediately. Finishing the questionnaire clears the deadline. They get a direct message, an optional warning, and a case if they are removed.

## Discord setup

1. Create an application and a bot at the Discord developer portal.
2. Turn on the **Server Members Intent** and the **Message Content Intent**.
3. Add this redirect: `{PUBLIC_URL}/auth/discord/callback`. The public Terms of Service are at `{PUBLIC_URL}/terms`. The public Privacy Policy is at `{PUBLIC_URL}/privacy`. Discord application verification asks for both URLs.
4. Sign in from the dashboard. Discord signs you in and lists servers where SquirrelBot is already installed and you have Manage Server, or a role saved under Access. Someone with Manage Server chooses those roles. Use Add a server when you want Discord to install the bot on another server you manage. The permission integer is `1099847429206` (view channels, send messages, embed links, attach files, read history, mention roles, manage messages, manage channels, add reactions, change nickname, kick, ban, manage roles, and moderate members). Change Nickname lets the bot use a display name on that server. Mention roles lets a confession tag the mod role. Attach files lets an approved confession keep an uploaded file. Add the server again if an older invite is missing either permission. The same install, without signing in, is:

```text
https://discord.com/oauth2/authorize?client_id=CLIENT_ID&permissions=1099847429206&integration_type=0&scope=bot%20applications.commands
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
| `/confession` | Send an anonymous confession for moderators to review |
| `/confession-review` `/confession-channel` | Choose the review channel or the public confessions channel |
| `/gate` | Show application status, or post the application message |

Prefix commands use the prefix from Welcome, default `!`. A saved command is a Lua 5.1 script. `reply` sends one message, and returning a string does the same when `reply` is not called. `liquid` fills `{{ author.name }}`, `{{ author.id }}`, `{{ author.display_name }}`, `{{ args }}`, `{{ guild.name }}`, and `{{ prefix }}`. `react` and `delete` act on the invoking message. `addRole` and `removeRole` only change roles selected for that command. Older commands that are plain text still send that text until you save them again.

## Run it locally

Copy `.env.example` to `.env` and fill in the Discord token, OAuth client id, OAuth secret, and a `SESSION_SECRET` of at least 32 characters. The process reads that file from the project directory. A variable already set in your shell wins over the file. `PUBLIC_URL` must be the origin you open in the browser. For dashboard development that is `http://localhost:5173`.

`make dev` starts Postgres, Air, and the dashboard at `http://localhost:5173`. Air rebuilds the bot when Go files, SQL migrations, or `.env` change. `make bot` and `make web` start those processes on their own, after `make db`.

`make build` writes `bin/squirrelbot` and `web/build`. `make image` builds `squirrelbot:latest` and the next patch version. `make test` runs the Go unit tests. `make test-e2e` runs the API suite against Postgres and a Discord stand-in, then the browser suite against the dashboard. It starts the Compose Postgres if nothing is listening on port 5432.

`make up` runs the bot and Postgres in Docker. In that mode the dashboard is served by the bot on port 8080, so set `PUBLIC_URL` to `http://localhost:8080`. `make down` stops those containers.

`go run ./cmd/squirrelbot` starts one run and exits when you stop it.

## Kubernetes

The chart is `deploy/helm/squirrelbot`. It runs one replica with the Recreate strategy, a read-only root filesystem, and a non-root user. Liveness is `/healthz`. Readiness is `/readyz`, which stays down until Postgres answers and the Discord gateway is ready.

`--set` and a private values file make the chart create the Secret. To use a Secret that already exists, set `existingSecret` to its name. The chart then mounts that Secret and does not create one. The keys are `DATABASE_URL`, `DISCORD_TOKEN`, `DISCORD_CLIENT_ID`, `DISCORD_CLIENT_SECRET`, and `SESSION_SECRET`.

`deploy/flux/squirrelbot-kustomization.yaml` is a Flux Kustomization for the published chart. It reconciles `deploy/flux/squirrelbot`. Create a Secret named `squirrelbot` in the `squirrelbot` namespace before the first reconcile, with those keys. Point `sourceRef` at your Flux GitRepository when it is not named `flux-system`. Pin `version` in the HelmRelease to stop automatic upgrades.

```sh
helm upgrade --install squirrelbot deploy/helm/squirrelbot \
  --set publicUrl=https://mod.example.com \
  --set secrets.databaseUrl='postgres://squirrelbot:PASSWORD@db:5432/squirrelbot' \
  --set secrets.discordToken=TOKEN \
  --set secrets.discordClientId=CLIENT_ID \
  --set secrets.discordClientSecret=CLIENT_SECRET \
  --set secrets.sessionSecret=AT_LEAST_32_RANDOM_CHARACTERS
```

Set `postgres.bundled=true` and `postgres.auth.password` to start a bundled Postgres StatefulSet. That database is for a single cluster and a small server. Use an external Postgres for anything you need to restore. `postgres.sslMode` sets the libpq mode on `DATABASE_URL` when the chart writes that URL. Bundled Postgres uses `disable`. An external database uses `require`. An `sslmode` already present in `secrets.databaseUrl` is left as written.

`make image` builds `squirrelbot:latest` and the next patch version from the repository root. The first build is one patch above the chart app version. Later builds increment the patch again. The last version is stored in `build/image-version`. Set `imagePullSecrets` when the registry is private.

## Releases

Every pull request merged into `main` is released. A pull request runs formatting, vet, unit tests with the race detector, end-to-end tests, and a Helm chart lint. It also builds the container image. The release workflow runs the tests again, then publishes the container image, the Helm chart, and the binary archives. Releases run one at a time, in merge order.

A label on the pull request picks the version bump:

| Label | Effect |
| --- | --- |
| `release:major` | v0.1.8 becomes v1.0.0 |
| `release:minor` | v0.1.8 becomes v0.2.0 |
| `release:patch` | v0.1.8 becomes v0.1.9, and so does a pull request with no release label |
| `release:none` | Merge without publishing a release |

The first release is `v0.1.0`. A `release:major` label on that first pull request publishes `v1.0.0`. A check on each pull request fails when it has more than one of these labels, and the check says what merging it will release.

A release is published from a merged pull request, or from a manual run of the Release workflow. Open Actions, choose Release, and run the workflow. The manual run chooses the bump.

To create the labels once:

```sh
gh label create release:major --color B60205 --description "Breaking change: bump the major version"
gh label create release:minor --color FBCA04 --description "New feature: bump the minor version"
gh label create release:patch --color 0E8A16 --description "Fix: bump the patch version (the default)"
gh label create release:none  --color CCCCCC --description "Merge without making a release"
```

The release tags the merge commit. It sets `version` and `appVersion` on the Helm chart it publishes. `deploy/helm/squirrelbot/Chart.yaml` on `main` stays as it was in that merge.

Each release includes:

- Archives for Linux and macOS, on amd64 and arm64. An archive contains `squirrelbot`, `web/build`, and `LICENSE`. Extract it and run `./squirrelbot` from that directory.
- The image `ghcr.io/boxingoctopuscreative/squirrelbot`, tagged with the version and with `latest`.
- The Helm chart, attached to the GitHub release and published to `oci://ghcr.io/boxingoctopuscreative/charts/squirrelbot`. Each release also carries an `index.yaml`, so Helm can add the release archive as a chart repository.

The chart pulls `ghcr.io/boxingoctopuscreative/squirrelbot`. `make image` builds the local name `squirrelbot`. Set `image.repository=squirrelbot` to use that image. Install the published chart with the same `--set` values as the local chart, and set `--version` to the release you want.

```sh
helm repo add squirrelbot https://github.com/BoxingOctopusCreative/SquirrelBot/releases/latest/download
helm repo update
helm upgrade --install squirrelbot squirrelbot/squirrelbot --version 0.1.0
```

CI sets the GHCR package visibility to match the repository.

## License

SquirrelBot is licensed under the Mozilla Public License 2.0. The terms are in [LICENSE](LICENSE).
