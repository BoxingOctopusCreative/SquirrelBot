<script lang="ts">
	import { Button, Card, Input, P, Textarea } from 'flowbite-svelte';
	import { untrack } from 'svelte';
	import { sendJSON } from '#lib/api';
	import ChannelSelect from '#lib/ChannelSelect.svelte';
	import Field from '#lib/Field.svelte';
	import Notice from '#lib/Notice.svelte';
	import PageIntro from '#lib/PageIntro.svelte';
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
				leaveChannelId: guild.leaveChannelId,
				leaveMessage: guild.leaveMessage,
				autoroleId: guild.autoroleId,
				prefix: guild.prefix
			});
			notice = 'Saved. Welcome and leave messages updated.';
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not save.';
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>Welcome · SquirrelBot</title></svelte:head>
<PageIntro kicker="Arrival" title="Welcome">
	<P>The mod and admin roles can use staff commands even when their Discord permissions are narrower. Add a role under Dashboard access when those people should open this dashboard too. Placeholders in the welcome and leave messages: {'{user}'} {'{username}'} {'{server}'} {'{membercount}'}.</P>
</PageIntro>
<Card size="xl" class="flex-col p-5">
	<form class="grid gap-4" onsubmit={(event) => { event.preventDefault(); save(); }}>
		<div class="grid gap-4 md:grid-cols-2">
			<Field label="Mod role"><RoleSelect roles={data.bundle.roles} bind:value={guild.modRoleId} /></Field>
			<Field label="Admin role"><RoleSelect roles={data.bundle.roles} bind:value={guild.adminRoleId} /></Field>
		</div>
		<Field label="Mod log channel"><ChannelSelect channels={data.bundle.channels} bind:value={guild.modLogChannelId} /></Field>
		<Field label="Welcome channel"><ChannelSelect channels={data.bundle.channels} bind:value={guild.welcomeChannelId} /></Field>
		<Field label="Welcome message"><Textarea bind:value={guild.welcomeMessage} maxlength={1500} rows={6} /></Field>
		<Field label="Leave channel"><ChannelSelect channels={data.bundle.channels} bind:value={guild.leaveChannelId} /></Field>
		<Field label="Leave message" hint="Sent when a member leaves. Empty sends nothing.">
			<Textarea bind:value={guild.leaveMessage} maxlength={1500} rows={4} />
		</Field>
		<Field label="Auto role" hint="Given on join.">
			<RoleSelect roles={data.bundle.roles} bind:value={guild.autoroleId} />
		</Field>
		<Field label="Custom command prefix"><Input bind:value={guild.prefix} maxlength={5} required /></Field>
		<div class="flex flex-wrap items-center gap-3">
			<Button type="submit" disabled={busy}>Save Welcome</Button>
			<Notice text={notice} />
		</div>
	</form>
</Card>
