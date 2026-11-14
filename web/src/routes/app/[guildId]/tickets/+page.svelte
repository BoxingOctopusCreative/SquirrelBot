<script lang="ts">
	import { refreshAll } from '$app/navigation';
	import { untrack } from 'svelte';
	import { sendJSON, when } from '#lib/api';
	import ChannelSelect from '#lib/ChannelSelect.svelte';
	import CreateChannel from '#lib/CreateChannel.svelte';
	import Field from '#lib/Field.svelte';
	import Notice from '#lib/Notice.svelte';
	import RoleSelect from '#lib/RoleSelect.svelte';
	import type { Bundle, Ticket, TicketConfig } from '#lib/types';

	let { data }: { data: { bundle: Bundle; open: Ticket[]; closed: Ticket[] } } = $props();

	let cfg = $state<TicketConfig>(untrack(() => ({ ...data.bundle.tickets })));
	let notice = $state('');
	let busy = $state(false);

	function discordLink(channelId: string): string {
		return `https://discord.com/channels/${data.bundle.guild.id}/${channelId}`;
	}

	function claimLine(item: Ticket): string {
		return item.claimerName ? `Claimed by ${item.claimerName}` : 'Unclaimed';
	}

	async function save(): Promise<boolean> {
		busy = true;
		notice = '';
		try {
			const saved = await sendJSON<TicketConfig>(
				`/api/guilds/${data.bundle.guild.id}/tickets`,
				'PUT',
				$state.snapshot(cfg)
			);
			cfg = { ...saved };
			notice = 'Saved. Support tickets updated.';
			await refreshAll();
			return true;
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not save.';
			return false;
		} finally {
			busy = false;
		}
	}

	async function postPanel() {
		const ok = await save();
		if (!ok) return;
		busy = true;
		try {
			await sendJSON(`/api/guilds/${data.bundle.guild.id}/tickets/panel`, 'POST');
			notice = 'Message posted.';
		} catch (err) {
			const why = err instanceof Error ? err.message : 'Could not post.';
			notice = `Saved, but the panel was not posted. ${why}`;
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>Support tickets · SquirrelBot</title></svelte:head>
<p class="kicker">Help</p>
<h1>Support tickets</h1>
<p class="sub">
	A member presses Open a ticket on the panel. A form asks for a subject and what they need. SquirrelBot opens a private channel for that member and the support role. Staff claim the ticket, then close it with a reason. Closing saves a transcript here and posts a copy in the transcript channel. Reopen puts the ticket back. Delete asks for the ticket number, then removes the channel.
</p>
<form
	class="panel"
	onsubmit={(event) => {
		event.preventDefault();
		save();
	}}
>
	<Field label="Panel channel" hint="The Open a ticket button is posted here. Members can read this channel.">
		<ChannelSelect channels={data.bundle.channels} bind:value={cfg.panelChannelId} />
	</Field>
	<CreateChannel
		guildId={data.bundle.guild.id}
		purpose="ticket-panel"
		label="Panel channel name"
		after="Save tickets to use it."
		oncreated={(channel) => {
			cfg.panelChannelId = channel.id;
		}}
	/>
	<Field label="Ticket category" hint="New tickets are channels under this category. A category holds 50 channels.">
		<ChannelSelect channels={data.bundle.categories} prefix="" bind:value={cfg.categoryId} />
	</Field>
	<CreateChannel
		guildId={data.bundle.guild.id}
		purpose="ticket-category"
		kind="category"
		label="Category name"
		after="Save tickets to use it."
		oncreated={(channel) => {
			cfg.categoryId = channel.id;
		}}
	/>
	<Field
		label="Support role"
		hint="This role can see every ticket, claim it, and close it. The mod role and admin role from Welcome can do that too."
	>
		<RoleSelect roles={data.bundle.roles} bind:value={cfg.supportRoleId} />
	</Field>
	<Field
		label="Transcript channel"
		hint="A copy of each closed ticket is posted here. Leave this empty to keep transcripts only in this dashboard. A channel you create here is hidden from members."
	>
		<ChannelSelect channels={data.bundle.channels} bind:value={cfg.logChannelId} />
	</Field>
	<CreateChannel
		guildId={data.bundle.guild.id}
		purpose="ticket-log"
		label="Transcript channel name"
		after="Save tickets to use it."
		oncreated={(channel) => {
			cfg.logChannelId = channel.id;
		}}
	/>
	<Field label="Panel title">
		<input bind:value={cfg.panelTitle} maxlength="80" required />
	</Field>
	<Field label="Panel message">
		<textarea bind:value={cfg.panelBody} maxlength="800" required></textarea>
	</Field>
	<div class="actions">
		<button type="submit" disabled={busy}>Save tickets</button>
		<button type="button" disabled={busy} onclick={postPanel}>Save and post the panel</button>
		<Notice text={notice} />
	</div>
</form>

<section class="list" aria-labelledby="open-tickets">
	<h2 id="open-tickets">Open tickets</h2>
	{#each data.open as item (item.id)}
		<article class="item">
			<header>
				<h3>Ticket {item.number}: {item.subject}</h3>
				<span class="meta">{when(item.createdAt)}</span>
			</header>
			<p>{item.body}</p>
			<p class="meta">{item.openerName} · {claimLine(item)} · Open</p>
			{#if item.channelId}
				<p><a href={discordLink(item.channelId)} target="_blank" rel="noreferrer">Open ticket {item.number} in Discord</a></p>
			{/if}
		</article>
	{:else}
		<p class="hint">No open tickets.</p>
	{/each}
</section>

<section class="list" aria-labelledby="closed-tickets">
	<h2 id="closed-tickets">Closed tickets</h2>
	{#each data.closed as item (item.id)}
		<article class="item">
			<header>
				<h3>Ticket {item.number}: {item.subject}</h3>
				<span class="meta">{when(item.closedAt || item.createdAt)}</span>
			</header>
			<p>{item.body}</p>
			<p class="meta">
				{item.openerName} · {item.status === 'deleted' ? 'Channel deleted' : 'Closed'}
				{#if item.closedByName}· {item.closedByName}{/if}
			</p>
			{#if item.closeReason}<p>{item.closeReason}</p>{/if}
			{#if item.channelId && item.status !== 'deleted'}
				<p><a href={discordLink(item.channelId)} target="_blank" rel="noreferrer">Open ticket {item.number} in Discord</a></p>
			{/if}
			{#if item.transcript}
				<details>
					<summary>Transcript for ticket {item.number}</summary>
					<pre>{item.transcript}</pre>
				</details>
			{:else}
				<p class="hint">No transcript was saved.</p>
			{/if}
		</article>
	{:else}
		<p class="hint">No closed tickets.</p>
	{/each}
</section>
