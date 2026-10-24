<script lang="ts">
	import { untrack } from 'svelte';
	import { sendJSON } from '#lib/api';
	import Field from '#lib/Field.svelte';
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
<p class="kicker">Filters</p>
<h1>Automod</h1>
<p class="sub">Deletes invites, repeated messages, mention piles, walls of caps, and words you block. Staff with Manage Messages, and the mod or admin role, are skipped.</p>
<form class="panel" onsubmit={(event) => { event.preventDefault(); saveAutomod(); }}>
	<label class="check"><input type="checkbox" bind:checked={automod.enabled} /> Automod is on</label>
	<label class="check"><input type="checkbox" bind:checked={automod.blockInvites} /> Block Discord invites</label>
	<label class="check"><input type="checkbox" bind:checked={automod.blockSpam} /> Block repeated messages</label>
	<div class="split">
		<Field label="Mention limit" hint="0 turns this check off."><input type="number" min="0" max="50" bind:value={automod.mentionLimit} /></Field>
		<Field label="Caps percent" hint="0 turns this check off."><input type="number" min="0" max="100" bind:value={automod.capsPercent} /></Field>
	</div>
	<Field label="Minimum letters before the caps check"><input type="number" min="0" max="200" bind:value={automod.capsMinLength} /></Field>
	<Field label="Blocked words" hint="One word per line. Whole words only."><textarea bind:value={words}></textarea></Field>
	<Field label="After deleting">
		<select bind:value={automod.action}>
			<option value="delete">Delete only</option>
			<option value="warn">Delete and warn</option>
			<option value="timeout">Delete and timeout</option>
			<option value="kick">Delete and kick</option>
		</select>
	</Field>
	<Field label="Timeout length, in seconds"><input type="number" min="60" max="2419200" bind:value={automod.timeoutSeconds} /></Field>
	<Field label="Ignored channels">
		<select multiple bind:value={automod.ignoredChannelIds}>
			{#each data.bundle.channels as channel (channel.id)}
				<option value={channel.id}>#{channel.name}</option>
			{/each}
		</select>
	</Field>
	<Field label="Ignored roles">
		<select multiple bind:value={automod.ignoredRoleIds}>
			{#each data.bundle.roles as role (role.id)}
				<option value={role.id}>{role.name}</option>
			{/each}
		</select>
	</Field>
	<div class="actions">
		<button type="submit" disabled={busy}>Save automod</button>
		{#if notice}<span class="banner">{notice}</span>{/if}
	</div>
</form>
