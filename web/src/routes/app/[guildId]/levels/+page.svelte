<script lang="ts">
	import { refreshAll } from '$app/navigation';
	import { Button, Card, Checkbox, Heading, P, Textarea } from 'flowbite-svelte';
	import { untrack } from 'svelte';
	import { sendJSON, when } from '#lib/api';
	import ChannelSelect from '#lib/ChannelSelect.svelte';
	import ChoiceSelect from '#lib/ChoiceSelect.svelte';
	import CreateChannel from '#lib/CreateChannel.svelte';
	import Field from '#lib/Field.svelte';
	import IdMultiSelect from '#lib/IdMultiSelect.svelte';
	import Notice from '#lib/Notice.svelte';
	import NumberInput from '#lib/NumberInput.svelte';
	import PageIntro from '#lib/PageIntro.svelte';
	import RoleSelect from '#lib/RoleSelect.svelte';
	import type { Bundle, LevelEvent, LevelReward, LevelSettings, LevelStanding } from '#lib/types';

	let { data }: { data: { bundle: Bundle; leaderboard: LevelStanding[]; events: LevelEvent[]; total: number } } =
		$props();

	let settings = $state<LevelSettings>(
		untrack(() => ({
			...data.bundle.levels,
			announce: data.bundle.levels.announce || 'current',
			announceChannelId: data.bundle.levels.announceChannelId ?? '',
			message: data.bundle.levels.message ?? '',
			ignoredChannelIds: [...(data.bundle.levels.ignoredChannelIds ?? [])],
			ignoredRoleIds: [...(data.bundle.levels.ignoredRoleIds ?? [])],
			rewards: [...(data.bundle.levels.rewards ?? [])]
		}))
	);
	let backupAt = $state(untrack(() => data.bundle.levels.backupAt ?? ''));
	let rewardLevel = $state(1);
	let rewardRole = $state('');
	let confirmReset = $state(false);
	let notice = $state('');
	let resetNotice = $state('');
	let busy = $state(false);

	const announceIn = [
		{ value: 'current', name: 'In the channel' },
		{ value: 'channel', name: 'A chosen channel' },
		{ value: 'dm', name: 'Direct message' },
		{ value: 'off', name: 'Off' }
	];

	function roleName(id: string): string {
		return data.bundle.roles.find((role) => role.id === id)?.name ?? id;
	}

	function applySaved(saved: LevelSettings) {
		settings = {
			...saved,
			announce: saved.announce || 'current',
			announceChannelId: saved.announceChannelId ?? '',
			message: saved.message ?? '',
			ignoredChannelIds: [...(saved.ignoredChannelIds ?? [])],
			ignoredRoleIds: [...(saved.ignoredRoleIds ?? [])],
			rewards: [...(saved.rewards ?? [])]
		};
		if (saved.backupAt) backupAt = saved.backupAt;
	}

	async function save() {
		busy = true;
		notice = '';
		try {
			const saved = await sendJSON<LevelSettings>(`/api/guilds/${data.bundle.guild.id}/levels`, 'PUT', {
				...$state.snapshot(settings),
				xpMin: Number(settings.xpMin),
				xpMax: Number(settings.xpMax),
				rateCount: Number(settings.rateCount),
				rateSeconds: Number(settings.rateSeconds),
				announceMinLevel: Number(settings.announceMinLevel),
				announceMultiple: Number(settings.announceMultiple),
				rewards: settings.rewards.map((reward) => ({ level: Number(reward.level), roleId: reward.roleId }))
			});
			applySaved(saved);
			notice = 'Saved. Levels updated.';
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not save.';
		} finally {
			busy = false;
		}
	}

	function addReward() {
		const level = Number(rewardLevel);
		if (!rewardRole) {
			notice = 'Pick a role for that reward.';
			return;
		}
		if (settings.rewards.some((reward) => reward.level === level)) {
			notice = `Level ${level} already has a reward.`;
			return;
		}
		const next: LevelReward[] = [...settings.rewards, { level, roleId: rewardRole }];
		next.sort((a, b) => a.level - b.level);
		settings.rewards = next;
		notice = '';
	}

	function removeReward(level: number) {
		settings.rewards = settings.rewards.filter((reward) => reward.level !== level);
	}

	async function resetAll() {
		busy = true;
		resetNotice = '';
		try {
			const result = await sendJSON<{ members: number }>(`/api/guilds/${data.bundle.guild.id}/levels/reset`, 'POST');
			backupAt = new Date().toISOString();
			confirmReset = false;
			resetNotice = `Cleared XP for ${result.members} members. That backup replaces the previous one.`;
			await refreshAll();
		} catch (err) {
			resetNotice = err instanceof Error ? err.message : 'Could not reset levels.';
		} finally {
			busy = false;
		}
	}

	async function restore() {
		busy = true;
		resetNotice = '';
		try {
			const result = await sendJSON<{ members: number }>(
				`/api/guilds/${data.bundle.guild.id}/levels/restore`,
				'POST'
			);
			resetNotice = `Restored XP for ${result.members} members.`;
			await refreshAll();
		} catch (err) {
			resetNotice = err instanceof Error ? err.message : 'Could not restore levels.';
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>Levels · SquirrelBot</title></svelte:head>
<PageIntro kicker="Progress" title="Levels">
	<P>
		Members earn XP by chatting. The default is 15 to 25 XP per message, and one grant per 60 seconds. Level 1 is 100
		XP, level 2 is 255, and level 10 is 4,675. That is the Mee6 curve Carl-bot uses. Voice XP, when it is on, uses the
		same roll once a minute while a member is in a voice channel with someone else. Deafened members and the AFK
		channel do not earn it. A prefix command does not grant XP, and neither does a message automod removes.
		<code>/rank</code> and <code>/leaderboard</code> are open to everyone. <code>/level</code> needs Manage Server.
	</P>
</PageIntro>
<Card size="xl" class="mb-6 flex-col p-5">
	<form class="grid gap-4" onsubmit={(event) => { event.preventDefault(); save(); }}>
		<Checkbox bind:checked={settings.enabled}>Levels on</Checkbox>
		{#if !settings.enabled}<P>Turn this on before members can earn XP.</P>{/if}
		<div class="grid gap-4 md:grid-cols-2">
			<Field label="Minimum XP" hint="From 15 to 100. The default roll is 15 to 25.">
				<NumberInput min={15} max={100} bind:value={settings.xpMin} required />
			</Field>
			<Field label="Maximum XP" hint="Each grant picks a number from the minimum through this.">
				<NumberInput min={15} max={100} bind:value={settings.xpMax} required />
			</Field>
		</div>
		<div class="grid gap-4 md:grid-cols-2">
			<Field label="Grants per window" hint="Carl-bot's rate 5 60 is 5 messages per 60 seconds. The default is 1.">
				<NumberInput min={1} max={20} bind:value={settings.rateCount} required />
			</Field>
			<Field label="Window, in seconds" hint="From 10 to 3600. The default is 60.">
				<NumberInput min={10} max={3600} bind:value={settings.rateSeconds} required />
			</Field>
		</div>
		<Checkbox bind:checked={settings.voiceXp}>Voice XP</Checkbox>
		<Field label="Channels that do not earn XP">
			<IdMultiSelect
				options={data.bundle.channels.map((channel) => ({ value: channel.id, name: `#${channel.name}` }))}
				bind:value={settings.ignoredChannelIds}
			/>
		</Field>
		<Field label="Roles that do not earn XP">
			<IdMultiSelect
				options={data.bundle.roles.map((role) => ({ value: role.id, name: role.name }))}
				bind:value={settings.ignoredRoleIds}
			/>
		</Field>
		<Field label="Level-up announcement">
			<ChoiceSelect options={announceIn} bind:value={settings.announce} />
		</Field>
		<Field label="Level-up channel" hint="Used when announcements go to a chosen channel. Voice and manual XP use it when the announcement is set to the channel they were in.">
			<ChannelSelect channels={data.bundle.channels} bind:value={settings.announceChannelId} />
		</Field>
		<CreateChannel
			guildId={data.bundle.guild.id}
			purpose="levels"
			after="Save the channel to use it."
			oncreated={(channel) => { settings.announceChannelId = channel.id; }}
		/>
		<Field label="Level-up message" hint="Leave this empty for the default. Variables: mention, user, level, server, xp.">
			<Textarea bind:value={settings.message} maxlength={500} rows={3} />
		</Field>
		<div class="grid gap-4 md:grid-cols-2">
			<Field label="Only announce above level" hint="0 announces every level. 15 announces 16 and up.">
				<NumberInput min={0} max={100} bind:value={settings.announceMinLevel} />
			</Field>
			<Field label="Only announce multiples of" hint="0 announces every level. 5 announces 5, 10, 15, and so on.">
				<NumberInput min={0} max={100} bind:value={settings.announceMultiple} />
			</Field>
		</div>
		<Checkbox bind:checked={settings.rewardOnly}>Only announce levels that grant a role</Checkbox>
		<Checkbox bind:checked={settings.replaceRewards}>Replace earlier reward roles</Checkbox>
		<div class="grid gap-2">
			<p class="text-sm font-medium text-gray-900 dark:text-white">Role rewards</p>
			<p class="text-sm text-gray-500 dark:text-gray-400">One role per level, from 1 to 100. Rewards update the next time that member's level changes. Save to keep a role you add or remove here.</p>
			{#if settings.rewards.length === 0}
				<P>No role rewards yet.</P>
			{:else}
				<ul class="list">
					{#each settings.rewards as reward (reward.level)}
						<li class="flex flex-wrap items-center justify-between gap-2">
							<span>Level {reward.level} · {roleName(reward.roleId)}</span>
							<Button type="button" color="alternative" size="sm" onclick={() => removeReward(reward.level)}>Remove</Button>
						</li>
					{/each}
				</ul>
			{/if}
		</div>
		<div class="grid gap-4 md:grid-cols-2">
			<Field label="Reward level">
				<NumberInput min={1} max={100} bind:value={rewardLevel} />
			</Field>
			<Field label="Reward role">
				<RoleSelect roles={data.bundle.roles} bind:value={rewardRole} emptyLabel="Choose a role" />
			</Field>
		</div>
		<Button type="button" color="alternative" onclick={addReward}>Add Reward</Button>
		<div class="flex flex-wrap items-center gap-3">
			<Button type="submit" disabled={busy}>Save Levels</Button>
			<Notice text={notice} />
		</div>
	</form>
</Card>
<Card size="xl" class="mb-6 flex-col gap-4 p-5">
	<Heading tag="h2" class="font-display text-xl font-semibold">Reset</Heading>
	<P>Reset all clears every score and keeps one backup. Restore puts that backup back. A second reset replaces the backup. Resetting one member is <code>/level reset</code>, and that member is not backed up.</P>
	<Checkbox bind:checked={confirmReset}>I understand this clears every member's XP</Checkbox>
	<div class="flex flex-wrap items-center gap-3">
		<Button type="button" color="red" disabled={busy || !confirmReset} onclick={resetAll}>Reset All Levels</Button>
		<Button type="button" color="alternative" disabled={busy || !backupAt} onclick={restore}>Restore Backup</Button>
		<Notice text={resetNotice} />
		{#if backupAt}<span class="text-sm text-gray-500 dark:text-gray-400">Last backup {when(backupAt)}</span>{/if}
	</div>
</Card>
<Card size="xl" class="flex-col gap-4 p-5">
	<Heading tag="h2" class="font-display text-xl font-semibold">Leaderboard</Heading>
	{#if data.total === 0}
		<P>No one has XP yet.</P>
	{:else}
		<P>{data.total} {data.total === 1 ? 'member has' : 'members have'} XP. This list is the first page.</P>
		<ul class="list">
			{#each data.leaderboard as row (row.userId)}
				<li>{row.rank}. {row.userName || row.userId} · level {row.level} · {row.xp} XP</li>
			{/each}
		</ul>
	{/if}
	<Heading tag="h2" class="font-display text-xl font-semibold">Recent Level Ups</Heading>
	{#if data.events.length === 0}
		<P>No level ups yet.</P>
	{:else}
		<ul class="list">
			{#each data.events as event, index (`${event.userId}-${event.createdAt}-${index}`)}
				<li>{when(event.createdAt)} · {event.userName || event.userId} reached level {event.level}</li>
			{/each}
		</ul>
	{/if}
</Card>
