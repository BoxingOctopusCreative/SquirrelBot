# Feature Specification: Timed Channel Purge

**Feature Branch**: `001-timed-channel-purge`

**Created**: 2026-11-27

**Status**: Draft

**Input**: User description: "This feature will allow moderators to automatically purge channels of all messages on a given schedule per channel. Schedules will be either pre-defined and common (1 hour, 4 hours, 24 hours, etc.) or can be custom. Like other features, this one gets a toggle in the admin panel."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Schedule a channel wipe on a common interval (Priority: P1)

An admin opens the server in the admin panel, turns Timed Channel Purge on, and attaches a common interval to a text channel. When that interval elapses, the channel loses every message Discord will still delete. The admin can see when the next wipe is due and what the last wipe did.

**Why this priority**: A repeating wipe on a known interval is the feature. Without it, the toggle and the custom interval have nothing to run.

**Independent Test**: Turn the feature on, set one text channel to 1 hour, wait for the interval, and confirm the eligible messages are gone and the panel shows the result.

**Acceptance Scenarios**:

1. **Given** Timed Channel Purge is off, **When** an admin turns it on and sets #lobby to 1 hour, **Then** the panel shows #lobby is scheduled and the next wipe is 1 hour after the save.
2. **Given** #lobby is scheduled for 1 hour and the feature is on, **When** the hour elapses, **Then** every message in #lobby that Discord will still delete is removed, and the panel shows when the wipe ran and how many messages were removed.
3. **Given** #lobby has messages older than 14 days and newer messages, **When** a scheduled wipe runs, **Then** the newer messages are removed, the older messages stay, and the panel counts the older messages as skipped.
4. **Given** an admin can change other server settings, **When** they open Timed Channel Purge, **Then** they can turn it on and save a schedule. A person who cannot change server settings cannot do either.
5. **Given** #lobby is scheduled, **When** an admin changes it from 1 hour to 24 hours, **Then** the next wipe is 24 hours after that save, and the channel is not wiped at the moment of the save.
6. **Given** #lobby contains a pinned message, a bot message, and a member message, all newer than 14 days, **When** the wipe runs, **Then** all three are removed and the channel has no new message announcing the wipe.

---

### User Story 2 - Set a custom interval (Priority: P2)

An admin needs a gap that is not in the common list, such as 90 minutes or 3 days. They enter that duration for one channel. The channel then wipes on that interval and no other channel changes.

**Why this priority**: The common list covers the usual cases. Custom intervals are the rest of the request, and they can ship after presets work.

**Independent Test**: With the feature on, save a 90 minute schedule on one channel and confirm the next wipe is 90 minutes out, while a second channel keeps its own schedule.

**Acceptance Scenarios**:

1. **Given** the feature is on and #lobby has no schedule, **When** an admin saves a custom interval of 90 minutes, **Then** the next wipe is 90 minutes after the save.
2. **Given** #lobby is on a 24 hour preset and #memes is on a 4 hour preset, **When** an admin sets #lobby to 3 days, **Then** #lobby uses 3 days and #memes stays on 4 hours.
3. **Given** a channel already has a schedule, **When** an admin enters 10 minutes or 31 days, **Then** the panel rejects the value and the saved schedule is unchanged.
4. **Given** an admin enters 60 minutes, **When** they save, **Then** the channel is scheduled for 60 minutes even though 1 hour is also a preset.

---

### User Story 3 - Stop wipes and read what happened (Priority: P3)

An admin pauses every scheduled wipe, or removes one channel, without losing the rest of the setup. After a wipe, or after a wipe that could not run, the panel shows the outcome on that channel.

**Why this priority**: Staff need a way to stop a wipe and a record of the last one. The preset schedule in story 1 is still useful before this polish, because the first run already reports a result.

**Independent Test**: Schedule two channels, remove one, turn the feature off, and confirm only the intended wipes stop. Force a channel the bot cannot clean and confirm the failure stays visible.

**Acceptance Scenarios**:

1. **Given** two channels are scheduled and the feature is on, **When** an admin removes the schedule from one channel, **Then** that channel has no further wipes and the other channel keeps its schedule.
2. **Given** channels are scheduled, **When** an admin turns Timed Channel Purge off, **Then** no new wipe starts, the schedules stay saved, and the panel shows the wipes as paused rather than due. Turning it back on does not wipe immediately. Each saved channel's next wipe is one full interval after the feature is turned back on.
3. **Given** a wipe is already deleting messages, **When** an admin turns the feature off, **Then** that wipe finishes and no later wipe starts until the feature is on again.
4. **Given** the bot cannot delete messages in a scheduled channel, **When** a wipe comes due, **Then** the schedule stays, the panel shows the failure, and the bot tries again at the next interval.
5. **Given** a scheduled channel has been deleted, **When** the next wipe comes due, **Then** the bot stops trying that channel and the panel shows the channel is gone.

---

### Edge Cases

- A channel with no messages, or with only messages older than 14 days, finishes as a successful wipe that removed nothing. It is not an error.
- Messages sent while a wipe is in progress can remain until the next interval. The result counts the messages that wipe tried to remove.
- If a wipe is still running when the next interval arrives, a second wipe for that channel does not start.
- If the bot is down when a wipe is due, each scheduled channel runs once after the bot is back. It does not run once for every missed interval.
- Two admins saving the same channel: the last save wins, and the wait restarts from that save.
- Pinned messages, bot messages, and member messages are all removed when Discord allows the delete. A wipe does not leave a new message in the channel.
- Threads and forum posts are not removed as a side effect of wiping the parent channel.
- The manual purge command keeps its current behavior. This feature does not add a "wipe now" action.
- Timed Channel Purge does not open a moderation case. It cleans a channel. It does not punish a member. Warns, kicks, bans, timeouts, and the other recorded punishments are unchanged.
- Voice channels and categories cannot be scheduled. Only text channels that hold messages can.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The admin panel MUST show a server-wide on/off control for Timed Channel Purge, in the same way other features can be turned on or off.
- **FR-002**: While the control is off, the bot MUST NOT start a scheduled wipe. Schedules already saved MUST remain.
- **FR-003**: Turning the control back on MUST NOT wipe a channel at that moment. Each saved channel's next wipe MUST be one full interval after the control is turned on.
- **FR-004**: An admin MUST be able to attach at most one schedule to each text channel. A new save MUST replace the previous schedule for that channel.
- **FR-005**: A schedule MUST be either one common interval or one custom duration.
- **FR-006**: The common intervals MUST be exactly 1 hour, 4 hours, 12 hours, 24 hours, and 7 days.
- **FR-007**: A custom duration MUST be a whole number of minutes from 15 through 43,200 (30 days). A value outside that range MUST be rejected and MUST NOT change the saved schedule.
- **FR-008**: Saving a new or changed schedule MUST set the next wipe to one full interval after the save, and MUST NOT wipe the channel at the moment of the save.
- **FR-009**: When a wipe comes due and the feature is on, the bot MUST remove every message in that channel that Discord still allows to be deleted.
- **FR-010**: Messages older than 14 days MUST stay in the channel. The result MUST count them as skipped.
- **FR-011**: A wipe MUST include member messages, bot messages, and pinned messages when Discord allows those deletes.
- **FR-012**: The bot MUST NOT post a message into the channel to announce the wipe.
- **FR-013**: The admin panel MUST show, for each scheduled channel, the interval, the next wipe time, the last wipe time, how many messages were removed, how many were skipped, and the last failure if the wipe did not complete. While the feature is off, the panel MUST show those schedules as paused and MUST NOT present them as due.
- **FR-014**: An admin MUST be able to remove one channel's schedule. That channel MUST have no further wipes. Other channels MUST be unchanged.
- **FR-015**: If a wipe for a channel is still in progress, the bot MUST NOT start another wipe for that same channel.
- **FR-016**: If the bot was unavailable when a wipe was due, it MUST wipe that channel once after it is available again, not once per missed interval.
- **FR-017**: A wipe that has already started MUST finish if an admin turns the feature off during the run. No later wipe starts until the feature is on again.
- **FR-018**: If the bot cannot delete messages in a scheduled channel, the schedule MUST stay, the panel MUST show the failure, and the bot MUST try again at the next interval.
- **FR-019**: If a scheduled channel no longer exists, the bot MUST stop trying it and the panel MUST show that the channel is gone.
- **FR-020**: A wipe of an empty channel, or of a channel whose messages are all older than 14 days, MUST complete as a success with zero messages removed.
- **FR-021**: Only people who can already change that server's settings in the admin panel MUST be able to turn Timed Channel Purge on or off, save a schedule, or remove a schedule.
- **FR-022**: The existing manual purge action MUST keep its current behavior.
- **FR-023**: Wiping a text channel MUST NOT also wipe its threads or forum posts.
- **FR-024**: A scheduled wipe MUST NOT create a moderation case.

### Key Entities

- **Timed purge control**: The server-wide on/off state. Off means saved schedules stay and do not run.
- **Channel schedule**: One text channel, either a common interval or a custom duration, the next wipe time, and the last result. A channel has at most one schedule.
- **Wipe result**: When the wipe ran, how many messages were removed, how many were skipped because they were older than 14 days, and the failure text if the wipe did not complete.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An admin who can change server settings can turn Timed Channel Purge on and save a common interval on a channel in under 2 minutes.
- **SC-002**: Eligible messages are removed within 5 minutes of the scheduled time. A channel with 1,000 eligible messages is clear of those messages within 10 minutes of the scheduled time.
- **SC-003**: On a finished wipe, the panel's removed count plus the skipped count accounts for every message that was in the channel when the wipe started.
- **SC-004**: While the feature is off, a due schedule removes no messages. Turning it back on does not remove messages until a full interval has passed.
- **SC-005**: Every custom interval under 15 minutes or over 30 days is rejected, and the previously saved schedule is still the one in effect.
- **SC-006**: After the bot was down across several due times, each scheduled channel is wiped once, and staff can see that single result on the panel.
- **SC-007**: In a trial with two scheduled channels, removing one schedule stops that channel's wipes and leaves the other channel's interval and next wipe time unchanged.

## Assumptions

- "Moderators" in the request means the staff who already manage the server. Schedule setup lives in the admin panel, for the same people who can change other features there. Moderators do not get a separate in-chat command for this feature.
- The common list is closed: 1 hour, 4 hours, 12 hours, 24 hours, and 7 days. The request named 1 hour, 4 hours, and 24 hours, and "etc." is filled by 12 hours and 7 days. Anything else is a custom duration.
- A custom schedule is a length of time, not a clock time. "Every day at 09:00" is out of scope. The wait starts when the schedule is saved, when the interval is changed, or when the feature is turned back on.
- The shortest custom interval is 15 minutes. The longest is 30 days. Shorter gaps would wipe a busy channel too often. Longer gaps belong outside this feature.
- "All messages" includes pinned messages and bot messages. There is no filter by author, keyword, or pin. Staff who need to keep a message should put it somewhere that is not scheduled.
- Discord refuses bulk deletion of messages older than 14 days. Timed wipes follow that same limit the manual purge already follows. Those messages stay, and the panel counts them.
- The bot does not leave a notice in the wiped channel. The admin panel is the record. A moderation case is the wrong record, because nobody was warned, kicked, banned, or timed out.
- This feature uses the bot's existing schedule and the server's existing saved settings. It does not add a second bot or a second scheduler. Saved schedules and due wipes survive a restart (Principle I. One Process). Who may edit them stays the same as other admin panel settings (Principle IV. Least Privilege). A wipe is not a punishment, so Principle III does not require a case.
- Voice channels, categories, direct messages, a "wipe now" button, and wiping threads or forum posts by wiping the parent are out of scope.
