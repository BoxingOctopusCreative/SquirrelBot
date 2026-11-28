<script lang="ts">
	import { Button, Card, Checkbox, P } from 'flowbite-svelte';
	import { untrack } from 'svelte';
	import { sendJSON, when } from '#lib/api';
	import ChannelSelect from '#lib/ChannelSelect.svelte';
	import ChoiceSelect from '#lib/ChoiceSelect.svelte';
	import Field from '#lib/Field.svelte';
	import Notice from '#lib/Notice.svelte';
	import NumberInput from '#lib/NumberInput.svelte';
	import PageIntro from '#lib/PageIntro.svelte';
	import type { Bundle, ChannelPurge } from '#lib/types';

	let { data }: { data: { bundle: Bundle; schedules: ChannelPurge[] } } = $props();

	let purgeOn = $state(untrack(() => data.bundle.guild.purgeEnabled));
	let schedules = $state<ChannelPurge[]>(untrack(() => data.schedules));
	let channelId = $state(untrack(() => data.bundle.channels[0]?.id ?? ''));
	let preset = $state('1h');
	let minutes = $state(90);
	let editId = $state('');
	let editPreset = $state('1h');
	let editMinutes = $state(90);
	let toggleNotice = $state('');
	let notice = $state('');
	let editNotice = $state('');
	let editNoticeId = $state('');
	let busy = $state(false);
	let editBusy = $state(false);

	const intervals = [
		{ value: '1h', name: '1 hour' },
		{ value: '4h', name: '4 hours' },
		{ value: '12h', name: '12 hours' },
		{ value: '24h', name: '24 hours' },
		{ value: '7d', name: '7 days' },
		{ value: 'custom', name: 'Custom' }
	];

	const presetLabels: Record<string, string> = {
		'1h': '1 hour',
		'4h': '4 hours',
		'12h': '12 hours',
		'24h': '24 hours',
		'7d': '7 days'
	};

	function intervalLabel(item: ChannelPurge): string {
		if (item.preset && presetLabels[item.preset]) return presetLabels[item.preset];
		return `${Math.round(item.intervalSeconds / 60)} minutes`;
	}

	function channelName(id: string): string {
		return data.bundle.channels.find((channel) => channel.id === id)?.name ?? id;
	}

	async function reload() {
		const listed = await sendJSON<{ enabled: boolean; schedules: ChannelPurge[] }>(
			`/api/guilds/${data.bundle.guild.id}/purges`,
			'GET'
		);
		purgeOn = listed.enabled;
		schedules = listed.schedules;
	}

	async function saveToggle() {
		toggleNotice = '';
		try {
			await sendJSON(`/api/guilds/${data.bundle.guild.id}/purges/enabled`, 'PUT', { enabled: purgeOn });
			await reload();
			toggleNotice = 'Saved. Timed purge updated.';
		} catch (err) {
			toggleNotice = err instanceof Error ? err.message : 'Could not save.';
		}
	}

	function edit(item: ChannelPurge) {
		editId = item.channelId;
		editNotice = '';
		editNoticeId = '';
		if (item.preset && presetLabels[item.preset]) {
			editPreset = item.preset;
		} else {
			editPreset = 'custom';
			editMinutes = Math.round(item.intervalSeconds / 60);
		}
	}

	async function saveSchedule() {
		busy = true;
		notice = '';
		try {
			const body =
				preset === 'custom'
					? { channelId, minutes: Number(minutes) }
					: { channelId, preset };
			await sendJSON(`/api/guilds/${data.bundle.guild.id}/purges`, 'POST', body);
			await reload();
			notice = 'Saved. Schedule updated.';
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not save.';
		} finally {
			busy = false;
		}
	}

	async function saveEdit() {
		editBusy = true;
		editNotice = '';
		try {
			const body =
				editPreset === 'custom'
					? { channelId: editId, minutes: Number(editMinutes) }
					: { channelId: editId, preset: editPreset };
			await sendJSON(`/api/guilds/${data.bundle.guild.id}/purges`, 'POST', body);
			const saved = editId;
			await reload();
			editId = '';
			editNoticeId = saved;
			editNotice = 'Saved. Schedule updated.';
		} catch (err) {
			editNoticeId = editId;
			editNotice = err instanceof Error ? err.message : 'Could not save.';
		} finally {
			editBusy = false;
		}
	}

	async function remove(id: string) {
		notice = '';
		try {
			await sendJSON(`/api/guilds/${data.bundle.guild.id}/purges/${id}`, 'DELETE');
			if (id === editId) editId = '';
			await reload();
			notice = 'Saved. Schedule removed.';
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not remove that schedule.';
		}
	}
</script>

<svelte:head><title>Timed Channel Purge · SquirrelBot</title></svelte:head>
<PageIntro kicker="Moderation" title="Timed Channel Purge">
	<P>Wipe a text channel on a repeating interval. Messages older than 14 days stay.</P>
</PageIntro>

<Card size="xl" class="mb-6 flex-col gap-4 p-5">
	<Checkbox bind:checked={purgeOn}>Timed purge on</Checkbox>
	{#if !purgeOn}
		<P>Turn this on to wipe scheduled channels. Saved schedules stay, and they do not run while this is off.</P>
	{/if}
	<div class="flex flex-wrap items-center gap-3">
		<Button type="button" onclick={saveToggle}>Save</Button>
		<Notice text={toggleNotice} />
	</div>
</Card>

<Card size="xl" class="mb-6 flex-col gap-4 p-5">
	<form
		class="grid gap-4"
		onsubmit={(event) => {
			event.preventDefault();
			saveSchedule();
		}}
	>
		<Field label="Channel">
			<ChannelSelect channels={data.bundle.channels} bind:value={channelId} allowEmpty={false} />
		</Field>
		<Field label="Interval">
			<ChoiceSelect options={intervals} bind:value={preset} />
		</Field>
		{#if preset === 'custom'}
			<Field label="Minutes" hint="From 15 minutes through 30 days.">
				<NumberInput bind:value={minutes} min={15} max={43200} required />
			</Field>
		{/if}
		<div class="flex flex-wrap items-center gap-3">
			<Button type="submit" disabled={busy || channelId === ''}>Save Schedule</Button>
			<Notice text={notice} />
		</div>
	</form>
</Card>

{#if schedules.length > 0}
	<ul class="grid gap-3">
		{#each schedules as item (item.channelId)}
			<li>
				<Card size="xl" class="flex-col gap-2 p-5">
					<P><b>#{channelName(item.channelId)}</b> · {intervalLabel(item)}</P>
					{#if item.state === 'paused'}
						<P>Paused</P>
					{:else if item.state === 'running'}
						<P>Wipe in progress</P>
					{:else if item.state === 'missing'}
						<P>That channel is gone.</P>
					{:else}
						<P>Next wipe: {when(item.nextRunAt)}</P>
					{/if}
					{#if item.lastRunAt}
						<P>Last wipe: {when(item.lastRunAt)}. Removed {item.removed}. Skipped {item.skipped}.</P>
					{/if}
					{#if item.error}<P>{item.error}</P>{/if}
					{#if editId === item.channelId}
						<form
							class="grid gap-4"
							onsubmit={(event) => {
								event.preventDefault();
								saveEdit();
							}}
						>
							<Field label="Interval">
								<ChoiceSelect options={intervals} bind:value={editPreset} />
							</Field>
							{#if editPreset === 'custom'}
								<Field label="Minutes" hint="From 15 minutes through 30 days.">
									<NumberInput bind:value={editMinutes} min={15} max={43200} required />
								</Field>
							{/if}
							<div class="flex flex-wrap items-center gap-3">
								<Button size="sm" type="submit" disabled={editBusy}>Update Schedule</Button>
								<Button color="alternative" size="sm" type="button" onclick={() => (editId = '')}>Cancel</Button>
							</div>
						</form>
					{/if}
					<div class="flex flex-wrap items-center gap-2">
						{#if editId !== item.channelId}
							<Button color="alternative" size="sm" type="button" onclick={() => edit(item)}>Edit</Button>
						{/if}
						<Button color="red" size="sm" type="button" onclick={() => remove(item.channelId)}>Remove</Button>
						{#if editNoticeId === item.channelId}<Notice text={editNotice} />{/if}
					</div>
				</Card>
			</li>
		{/each}
	</ul>
{/if}
