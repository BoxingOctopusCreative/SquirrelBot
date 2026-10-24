<script lang="ts">
	import { refreshAll } from '$app/navigation';
	import { untrack } from 'svelte';
	import { sendJSON } from '#lib/api';
	import ChannelSelect from '#lib/ChannelSelect.svelte';
	import CreateChannel from '#lib/CreateChannel.svelte';
	import CreateRole from '#lib/CreateRole.svelte';
	import Field from '#lib/Field.svelte';
	import RoleSelect from '#lib/RoleSelect.svelte';
	import type { Bundle, ReactionRole } from '#lib/types';

	let { data }: { data: { bundle: Bundle; reactionRoles: ReactionRole[] } } = $props();
	let channelId = $state(
		untrack(() => data.bundle.guild.reactionChannelId || data.reactionRoles[0]?.channelId || '')
	);
	let emoji = $state('');
	let roleId = $state('');
	let notice = $state('');
	let busy = $state(false);

	function roleName(id: string): string {
		return data.bundle.roles.find((role) => role.id === id)?.name ?? id;
	}

	function channelPath(): string {
		return `/api/guilds/${data.bundle.guild.id}/reaction-roles/channel`;
	}

	async function saveChannel() {
		if (!channelId) {
			notice = 'Pick a roles channel.';
			return;
		}
		busy = true;
		notice = '';
		try {
			await sendJSON(channelPath(), 'PUT', { channelId });
			notice = 'Roles channel saved.';
			await refreshAll();
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not save the channel.';
		} finally {
			busy = false;
		}
	}

	async function add(event: SubmitEvent) {
		event.preventDefault();
		notice = '';
		if (!channelId) {
			notice = 'Pick a roles channel.';
			return;
		}
		busy = true;
		try {
			if (channelId !== data.bundle.guild.reactionChannelId) {
				await sendJSON(channelPath(), 'PUT', { channelId });
			}
			await sendJSON(`/api/guilds/${data.bundle.guild.id}/reaction-roles`, 'POST', { emoji, roleId });
			emoji = '';
			notice = 'Saved. The bot posted the reaction.';
			await refreshAll();
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not save.';
		} finally {
			busy = false;
		}
	}

	async function remove(id: number) {
		notice = '';
		try {
			await sendJSON(`/api/guilds/${data.bundle.guild.id}/reaction-roles/${id}`, 'DELETE');
			await refreshAll();
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not delete.';
		}
	}
</script>

<svelte:head><title>Reaction roles · SquirrelBot</title></svelte:head>
<p class="kicker">Roles</p>
<h1>Reaction roles</h1>
<p class="sub">
	Pick the channel where members choose roles. Click a reaction to get the role. Remove the reaction to drop it.
	For a custom emoji, use name:id. SquirrelBot posts the message.
</p>
<form class="panel" onsubmit={add}>
	<Field label="Roles channel" hint="SquirrelBot posts one message in this channel.">
		<ChannelSelect channels={data.bundle.channels} bind:value={channelId} emptyLabel="Choose a channel" />
	</Field>
	<Field label="Emoji"><input bind:value={emoji} required /></Field>
	<Field label="Role" hint="Pick a role, or create one.">
		<RoleSelect roles={data.bundle.roles} bind:value={roleId} allowEmpty={false} />
	</Field>
	<div class="actions">
		<button type="submit" disabled={busy}>Add reaction role</button>
		<CreateRole
			guildId={data.bundle.guild.id}
			oncreated={(role) => {
				roleId = role.id;
			}}
		/>
		<CreateChannel
			guildId={data.bundle.guild.id}
			purpose="roles"
			after="Save the channel to use it."
			oncreated={(channel) => {
				channelId = channel.id;
			}}
		/>
		{#if channelId && channelId !== data.bundle.guild.reactionChannelId}
			<button class="quiet" type="button" disabled={busy} onclick={saveChannel}>Save channel</button>
		{/if}
		{#if notice}<span class="banner">{notice}</span>{/if}
	</div>
</form>
<section class="list">
	{#if data.bundle.guild.reactionMessageId}
		<p class="hint">
			<a href={`https://discord.com/channels/${data.bundle.guild.id}/${data.bundle.guild.reactionChannelId}/${data.bundle.guild.reactionMessageId}`}>Open the roles message</a>
		</p>
	{/if}
	{#each data.reactionRoles as item (item.id)}
		<article class="item">
			<header>
				<strong>{item.emoji} · {roleName(item.roleId)}</strong>
			</header>
			<div class="actions">
				<button class="quiet" type="button" onclick={() => remove(item.id)}>Delete</button>
			</div>
		</article>
	{:else}
		<p class="hint">No reaction roles yet.</p>
	{/each}
</section>
