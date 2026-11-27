<script lang="ts">
	import { A, Button, Card, Checkbox, Input, P } from 'flowbite-svelte';
	import { refreshAll } from '$app/navigation';
	import { untrack } from 'svelte';
	import { sendJSON } from '#lib/api';
	import ChannelSelect from '#lib/ChannelSelect.svelte';
	import CreateChannel from '#lib/CreateChannel.svelte';
	import CreateRole from '#lib/CreateRole.svelte';
	import Field from '#lib/Field.svelte';
	import Notice from '#lib/Notice.svelte';
	import PageIntro from '#lib/PageIntro.svelte';
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
	let reactionsOn = $state(untrack(() => data.bundle.guild.reactionsEnabled));
	let toggleNotice = $state('');

	function roleName(id: string): string {
		return data.bundle.roles.find((role) => role.id === id)?.name ?? id;
	}

	function channelPath(): string {
		return `/api/guilds/${data.bundle.guild.id}/reaction-roles/channel`;
	}

	async function saveToggle() {
		toggleNotice = '';
		try {
			await sendJSON(`/api/guilds/${data.bundle.guild.id}/reaction-roles/enabled`, 'PUT', {
				enabled: reactionsOn
			});
			toggleNotice = 'Saved. Reaction roles updated.';
			await refreshAll();
		} catch (err) {
			toggleNotice = err instanceof Error ? err.message : 'Could not save.';
		}
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

<svelte:head><title>Reaction Roles · SquirrelBot</title></svelte:head>
<PageIntro kicker="Roles" title="Reaction Roles">
	<P>
		Pick the channel where members choose roles. Click a reaction to get the role. Remove the reaction to drop it.
		For a custom emoji, use name:id. SquirrelBot posts the message.
	</P>
</PageIntro>
<Card size="xl" class="mb-6 flex-col gap-4 p-5">
	<Checkbox bind:checked={reactionsOn}>Reaction roles on</Checkbox>
	{#if !reactionsOn}<P>Turn this on before a reaction gives or removes a role. Starboard reactions keep working.</P>{/if}
	<div class="flex flex-wrap items-center gap-3">
		<Button type="button" onclick={saveToggle}>Save</Button>
		<Notice text={toggleNotice} />
	</div>
</Card>
<Card size="xl" class="mb-6 flex-col p-5">
	<form class="grid gap-4" onsubmit={add}>
		<Field label="Roles channel" hint="SquirrelBot posts one message in this channel.">
			<ChannelSelect channels={data.bundle.channels} bind:value={channelId} emptyLabel="Choose a channel" />
		</Field>
		<Field label="Emoji"><Input bind:value={emoji} required /></Field>
		<Field label="Role" hint="Pick a role, or create one.">
			<RoleSelect roles={data.bundle.roles} bind:value={roleId} allowEmpty={false} />
		</Field>
		<div class="flex flex-wrap items-center gap-3">
			<Button type="submit" disabled={busy}>Add Reaction Role</Button>
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
				<Button color="alternative" type="button" disabled={busy} onclick={saveChannel}>Save Channel</Button>
			{/if}
			<Notice text={notice} />
		</div>
	</form>
</Card>
<section class="list grid gap-3">
	{#if data.bundle.guild.reactionMessageId}
		<P class="text-sm">
			<A href={`https://discord.com/channels/${data.bundle.guild.id}/${data.bundle.guild.reactionChannelId}/${data.bundle.guild.reactionMessageId}`}>Open the roles message</A>
		</P>
	{/if}
	{#each data.reactionRoles as item (item.id)}
		<Card size="xl" class="flex-row items-center justify-between gap-3 p-4">
			<strong>{item.emoji} · {roleName(item.roleId)}</strong>
			<Button color="red" size="sm" type="button" onclick={() => remove(item.id)}>Delete</Button>
		</Card>
	{:else}
		<P class="text-sm">No reaction roles yet.</P>
	{/each}
</section>
