<script lang="ts">
	import { Button, Card, Checkbox, P, Select, Textarea } from 'flowbite-svelte';
	import { untrack } from 'svelte';
	import { sendJSON } from '#lib/api';
	import Field from '#lib/Field.svelte';
	import Notice from '#lib/Notice.svelte';
	import NumberInput from '#lib/NumberInput.svelte';
	import PageIntro from '#lib/PageIntro.svelte';
	import type { Automod, Bundle } from '#lib/types';

	let { data }: { data: { bundle: Bundle } } = $props();
	let automod = $state<Automod>(untrack(() => ({
		...data.bundle.automod,
		badWords: [...data.bundle.automod.badWords],
		ignoredChannelIds: [...data.bundle.automod.ignoredChannelIds],
		ignoredRoleIds: [...data.bundle.automod.ignoredRoleIds]
	})));
	let words = $state(untrack(() => data.bundle.automod.badWords.join('\n')));
	let notice = $state('');
	let busy = $state(false);

	const actions = [
		{ value: 'delete', name: 'Delete only' },
		{ value: 'warn', name: 'Delete and warn' },
		{ value: 'timeout', name: 'Delete and timeout' },
		{ value: 'kick', name: 'Delete and kick' }
	];

	async function saveAutomod() {
		busy = true;
		notice = '';
		try {
			await sendJSON(`/api/guilds/${data.bundle.guild.id}/automod`, 'PUT', {
				...automod,
				badWords: words.split('\n')
			});
			notice = 'Automod saved.';
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not save.';
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>Automod · SquirrelBot</title></svelte:head>
<PageIntro kicker="Filters" title="Automod">
	<P>Deletes invites, repeated messages, mention piles, walls of caps, and words you block. Staff with Manage Messages, and the mod or admin role, are skipped.</P>
</PageIntro>
<Card size="xl" class="flex-col p-5">
	<form class="grid gap-4" onsubmit={(event) => { event.preventDefault(); saveAutomod(); }}>
		<Checkbox bind:checked={automod.enabled}>Automod is on</Checkbox>
		<Checkbox bind:checked={automod.blockInvites}>Block Discord invites</Checkbox>
		<Checkbox bind:checked={automod.blockSpam}>Block repeated messages</Checkbox>
		<div class="grid gap-4 md:grid-cols-2">
			<Field label="Mention limit" hint="0 turns this check off."><NumberInput min={0} max={50} bind:value={automod.mentionLimit} /></Field>
			<Field label="Caps percent" hint="0 turns this check off."><NumberInput min={0} max={100} bind:value={automod.capsPercent} /></Field>
		</div>
		<Field label="Minimum letters before the caps check"><NumberInput min={0} max={200} bind:value={automod.capsMinLength} /></Field>
		<Field label="Blocked words" hint="One word per line. Whole words only."><Textarea bind:value={words} rows={6} /></Field>
		<Field label="After deleting">
			<Select items={actions} bind:value={automod.action} placeholder="" />
		</Field>
		<Field label="Timeout length, in seconds"><NumberInput min={60} max={2419200} bind:value={automod.timeoutSeconds} /></Field>
		<Field label="Ignored channels">
			<Select multiple placeholder="" bind:value={automod.ignoredChannelIds}>
				{#each data.bundle.channels as channel (channel.id)}
					<option value={channel.id}>#{channel.name}</option>
				{/each}
			</Select>
		</Field>
		<Field label="Ignored roles">
			<Select multiple placeholder="" bind:value={automod.ignoredRoleIds}>
				{#each data.bundle.roles as role (role.id)}
					<option value={role.id}>{role.name}</option>
				{/each}
			</Select>
		</Field>
		<div class="flex flex-wrap items-center gap-3">
			<Button type="submit" disabled={busy}>Save Automod</Button>
			<Notice text={notice} />
		</div>
	</form>
</Card>
