<script lang="ts">
	import { Button, Card, P } from 'flowbite-svelte';
	import { when } from '#lib/api';
	import PageIntro from '#lib/PageIntro.svelte';
	import type { Bundle } from '#lib/types';

	let { data }: { data: { bundle: Bundle } } = $props();
	let counts = $derived(data.bundle.counts);

	let sentence = $derived.by(() => {
		const parts: string[] = [];
		if (counts.pendingGate > 0) parts.push(`${counts.pendingGate} people still need to finish the application`);
		if (counts.openTickets === 1) parts.push('1 support ticket is open');
		else if (counts.openTickets > 0) parts.push(`${counts.openTickets} support tickets are open`);
		if (counts.activeEscalations > 0) parts.push(`${counts.activeEscalations} admin firecall grants are active`);
		if (counts.scheduledAnnouncements > 0) parts.push(`${counts.scheduledAnnouncements} announcements are waiting to post`);
		if (parts.length === 0) return 'Nothing is waiting on you.';
		return parts.join('. ') + '.';
	});

	let id = $derived(data.bundle.guild.id);
</script>

<svelte:head><title>{data.bundle.guild.name} · SquirrelBot</title></svelte:head>
<PageIntro kicker={data.bundle.guild.name} title="On duty">
	<P>{sentence}</P>
</PageIntro>
<nav class="mb-6 grid grid-cols-[repeat(auto-fit,minmax(9rem,1fr))] gap-3" aria-label="Live work">
	<Card href={`/app/${id}/announcements`} size="xl" class="flex-col justify-end gap-1 p-4 {counts.scheduledAnnouncements > 0 ? 'border-primary-500' : ''}">
		<b class="font-display text-3xl">{counts.scheduledAnnouncements}</b><span class="text-xs tracking-wide uppercase underline">Queued</span>
	</Card>
	<Card href={`/app/${id}/escalations`} size="xl" class="flex-col justify-end gap-1 p-4 {counts.activeEscalations > 0 ? 'border-primary-500' : ''}">
		<b class="font-display text-3xl">{counts.activeEscalations}</b><span class="text-xs tracking-wide uppercase underline">On Duty</span>
	</Card>
	<Card href={`/app/${id}/verification`} size="xl" class="flex-col justify-end gap-1 p-4 {counts.pendingGate > 0 ? 'border-red-500' : ''}">
		<b class="font-display text-3xl">{counts.pendingGate}</b><span class="text-xs tracking-wide uppercase underline">Applicants</span>
	</Card>
	<Card href={`/app/${id}/tickets`} size="xl" class="flex-col justify-end gap-1 p-4 {counts.openTickets > 0 ? 'border-green-500' : ''}">
		<b class="font-display text-3xl">{counts.openTickets}</b><span class="text-xs tracking-wide uppercase underline">Tickets</span>
	</Card>
</nav>
{#if data.bundle.nextKick}
	<P class="mb-4">Next application kick: {when(data.bundle.nextKick)}</P>
{/if}
<Card size="xl" class="flex-col gap-4 p-5">
	<P>{counts.cases} moderation cases on record.</P>
	<div class="flex flex-wrap items-center gap-2">
		<Button href={`/app/${id}/announcements`}>Write an announcement</Button>
		<Button href={`/app/${id}/escalations`} color="alternative">Grant admin firecall access</Button>
		<Button href={`/app/${id}/verification`} color="alternative">Set the application deadline</Button>
	</div>
</Card>
