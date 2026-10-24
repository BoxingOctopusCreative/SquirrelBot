<script lang="ts">
	import { untrack } from 'svelte';
	import { sendJSON } from '#lib/api';
	import ChannelSelect from '#lib/ChannelSelect.svelte';
	import Field from '#lib/Field.svelte';
	import RoleSelect from '#lib/RoleSelect.svelte';
	import type { Bundle, Guild } from '#lib/types';

	let { data }: { data: { bundle: Bundle } } = $props();
	let guild = $state<Guild>(untrack(() => ({ ...data.bundle.guild })));
	let notice = $state('');
	let busy = $state(false);

	async function save() {
		busy = true;
		notice = '';
		try {
			await sendJSON(`/api/guilds/${guild.id}`, 'PATCH', {
				modLogChannelId: guild.modLogChannelId,
				modRoleId: guild.modRoleId,
				adminRoleId: guild.adminRoleId,
				welcomeChannelId: guild.welcomeChannelId,
				welcomeMessage: guild.welcomeMessage,
				autoroleId: guild.autoroleId,
				prefix: guild.prefix
			});
			notice = 'Saved.';
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not save.';
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>Welcome · SquirrelBot</title></svelte:head>
<p class="kicker">Arrival</p>
<h1>Welcome</h1>
<p class="sub">The mod and admin roles can use staff commands even when their Discord permissions are narrower. Placeholders in the welcome message: {'{user}'} {'{username}'} {'{server}'} {'{membercount}'}.</p>
<form class="panel" onsubmit={(event) => { event.preventDefault(); save(); }}>
	<div class="split">
		<Field label="Mod role"><RoleSelect roles={data.bundle.roles} bind:value={guild.modRoleId} /></Field>
		<Field label="Admin role"><RoleSelect roles={data.bundle.roles} bind:value={guild.adminRoleId} /></Field>
	</div>
	<Field label="Mod log channel"><ChannelSelect channels={data.bundle.channels} bind:value={guild.modLogChannelId} /></Field>
	<Field label="Welcome channel"><ChannelSelect channels={data.bundle.channels} bind:value={guild.welcomeChannelId} /></Field>
	<Field label="Welcome message"><textarea bind:value={guild.welcomeMessage} maxlength="1500"></textarea></Field>
	<Field label="Auto role" hint="Given on join.">
		<RoleSelect roles={data.bundle.roles} bind:value={guild.autoroleId} />
	</Field>
	<Field label="Custom command prefix"><input bind:value={guild.prefix} maxlength="5" required /></Field>
	<div class="actions">
		<button type="submit" disabled={busy}>Save welcome</button>
		{#if notice}<span class="banner">{notice}</span>{/if}
	</div>
</form>
