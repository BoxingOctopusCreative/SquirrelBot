<script lang="ts">
	import { A, Button, Card, Heading, Input, P, Textarea } from 'flowbite-svelte';
	import { refreshAll } from '$app/navigation';
	import { untrack } from 'svelte';
	import { sendJSON, when } from '#lib/api';
	import ChannelSelect from '#lib/ChannelSelect.svelte';
	import CreateChannel from '#lib/CreateChannel.svelte';
	import Field from '#lib/Field.svelte';
	import Notice from '#lib/Notice.svelte';
	import PageIntro from '#lib/PageIntro.svelte';
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
<PageIntro kicker="Help" title="Support tickets">
	<P>
		A member presses Open a ticket on the panel. A form asks for a subject and what they need. SquirrelBot opens a private channel for that member and the support role. Staff claim the ticket, then close it with a reason. Closing saves a transcript here, posts a copy in the transcript channel, and deletes the ticket channel ten seconds later. Reopen before then puts the ticket back.
	</P>
</PageIntro>
<Card size="xl" class="mb-6 flex-col p-5">
	<form
		class="grid gap-4"
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
			<Input bind:value={cfg.panelTitle} maxlength={80} required />
		</Field>
		<Field label="Panel message">
			<Textarea bind:value={cfg.panelBody} maxlength={800} required rows={6} />
		</Field>
		<div class="flex flex-wrap items-center gap-3">
			<Button type="submit" disabled={busy}>Save tickets</Button>
			<Button type="button" disabled={busy} onclick={postPanel}>Save and post the panel</Button>
			<Notice text={notice} />
		</div>
	</form>
</Card>

<section class="list mb-6 grid gap-3" aria-labelledby="open-tickets">
	<Heading id="open-tickets" tag="h2" class="font-display text-xl font-semibold">Open tickets</Heading>
	{#each data.open as item (item.id)}
		<Card size="xl" class="flex-col gap-2 p-4">
			<header class="flex items-baseline justify-between gap-3">
				<Heading tag="h3" class="font-display text-lg font-semibold">Ticket {item.number}: {item.subject}</Heading>
				<span class="font-mono text-sm text-gray-500">{when(item.createdAt)}</span>
			</header>
			<P>{item.body}</P>
			<P class="font-mono text-sm">{item.openerName} · {claimLine(item)} · Open</P>
			{#if item.channelId}
				<P><A href={discordLink(item.channelId)} target="_blank" rel="noreferrer">Open ticket {item.number} in Discord</A></P>
			{/if}
		</Card>
	{:else}
		<P class="text-sm">No open tickets.</P>
	{/each}
</section>

<section class="list grid gap-3" aria-labelledby="closed-tickets">
	<Heading id="closed-tickets" tag="h2" class="font-display text-xl font-semibold">Closed tickets</Heading>
	{#each data.closed as item (item.id)}
		<Card size="xl" class="flex-col gap-2 p-4">
			<header class="flex items-baseline justify-between gap-3">
				<Heading tag="h3" class="font-display text-lg font-semibold">Ticket {item.number}: {item.subject}</Heading>
				<span class="font-mono text-sm text-gray-500">{when(item.closedAt || item.createdAt)}</span>
			</header>
			<P>{item.body}</P>
			<P class="font-mono text-sm">
				{item.openerName} · {item.status === 'deleted' ? 'Channel deleted' : 'Closed'}
				{#if item.closedByName}· {item.closedByName}{/if}
			</P>
			{#if item.closeReason}<P>{item.closeReason}</P>{/if}
			{#if item.channelId && item.status !== 'deleted'}
				<P><A href={discordLink(item.channelId)} target="_blank" rel="noreferrer">Open ticket {item.number} in Discord</A></P>
			{/if}
			{#if item.transcript}
				<details class="rounded-lg border border-gray-200 p-3 dark:border-gray-600">
					<summary class="cursor-pointer font-medium">Transcript for ticket {item.number}</summary>
					<pre class="mt-3 font-mono text-sm whitespace-pre-wrap">{item.transcript}</pre>
				</details>
			{:else}
				<P class="text-sm">No transcript was saved.</P>
			{/if}
		</Card>
	{:else}
		<P class="text-sm">No closed tickets.</P>
	{/each}
</section>
