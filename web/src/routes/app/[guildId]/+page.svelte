<script lang="ts">
	import { when } from '#lib/api';
	import type { Bundle } from '#lib/types';

	let { data }: { data: { bundle: Bundle } } = $props();
	let counts = $derived(data.bundle.counts);

	let sentence = $derived.by(() => {
		const parts: string[] = [];
		if (counts.pendingGate > 0) parts.push(`${counts.pendingGate} people still need to finish the application`);
		if (counts.activeEscalations > 0) parts.push(`${counts.activeEscalations} admin firecall grants are active`);
		if (counts.scheduledAnnouncements > 0) parts.push(`${counts.scheduledAnnouncements} announcements are waiting to post`);
		if (parts.length === 0) return 'Nothing is waiting on you.';
		return parts.join('. ') + '.';
	});
</script>

<svelte:head><title>{data.bundle.guild.name} · SquirrelBot</title></svelte:head>
<p class="kicker">{data.bundle.guild.name}</p>
<h1>On duty</h1>
<p class="sub">{sentence}</p>
<nav class="metrics" aria-label="Live work">
	<a class={counts.scheduledAnnouncements > 0 ? 'hot' : ''} style:--tone="#d08a4a" href={`/app/${data.bundle.guild.id}/announcements`}>
		<b>{counts.scheduledAnnouncements}</b><span>Queued</span>
	</a>
	<a class={counts.activeEscalations > 0 ? 'hot' : ''} style:--tone="#d08a4a" href={`/app/${data.bundle.guild.id}/escalations`}>
		<b>{counts.activeEscalations}</b><span>On Duty</span>
	</a>
	<a class={counts.pendingGate > 0 ? 'hot' : ''} style:--tone="#e15d52" href={`/app/${data.bundle.guild.id}/verification`}>
		<b>{counts.pendingGate}</b><span>Applicants</span>
	</a>
</nav>
{#if data.bundle.nextKick}
	<p class="sub">Next application kick: {when(data.bundle.nextKick)}</p>
{/if}
<section class="panel">
	<p>{counts.cases} moderation cases on record.</p>
	<div class="actions">
		<a class="button" href={`/app/${data.bundle.guild.id}/announcements`}>Write an announcement</a>
		<a class="button quiet" href={`/app/${data.bundle.guild.id}/escalations`}>Grant admin firecall access</a>
		<a class="button quiet" href={`/app/${data.bundle.guild.id}/verification`}>Set the application deadline</a>
	</div>
</section>
