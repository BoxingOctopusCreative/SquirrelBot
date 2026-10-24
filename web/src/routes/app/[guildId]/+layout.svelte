<script lang="ts">
	import { page } from '$app/state';
	import type { Snippet } from 'svelte';
	import type { Bundle } from '#lib/types';

	let { data, children }: { data: { bundle: Bundle }; children: Snippet } = $props();

	let id = $derived(data.bundle.guild.id);
	let counts = $derived(data.bundle.counts);
	let path = $derived(page.url.pathname.replace(/\/$/, ''));

	function current(href: string): 'page' | undefined {
		return path === href.replace(/\/$/, '') ? 'page' : undefined;
	}
</script>

<div class="workspace">
	<nav class="duty" aria-label="Live work">
		<a class={counts.scheduledAnnouncements > 0 ? 'hot' : ''} style:--tone="#d08a4a" href={`/app/${id}/announcements`}>
			<b>{counts.scheduledAnnouncements}</b><span>Queued</span>
		</a>
		<a class={counts.activeEscalations > 0 ? 'hot' : ''} style:--tone="#d08a4a" href={`/app/${id}/escalations`}>
			<b>{counts.activeEscalations}</b><span>On Duty</span>
		</a>
		<a class={counts.pendingGate > 0 ? 'hot' : ''} style:--tone="#e15d52" href={`/app/${id}/verification`}>
			<b>{counts.pendingGate}</b><span>Applicants</span>
		</a>
	</nav>
	<nav class="nav" aria-label="Server sections">
		<a href={`/app/${id}`} aria-current={current(`/app/${id}`)}>Overview</a>
		<a href={`/app/${id}/announcements`} aria-current={current(`/app/${id}/announcements`)}>Announcements</a>
		<a href={`/app/${id}/escalations`} aria-current={current(`/app/${id}/escalations`)}>Admin Firecall Access</a>
		<a href={`/app/${id}/verification`} aria-current={current(`/app/${id}/verification`)}>Server Application</a>
		<a href={`/app/${id}/cases`} aria-current={current(`/app/${id}/cases`)}>Cases</a>
		<a href={`/app/${id}/automod`} aria-current={current(`/app/${id}/automod`)}>Automod</a>
		<a href={`/app/${id}/logging`} aria-current={current(`/app/${id}/logging`)}>Logging</a>
		<a href={`/app/${id}/welcome`} aria-current={current(`/app/${id}/welcome`)}>Welcome</a>
		<a href={`/app/${id}/reactions`} aria-current={current(`/app/${id}/reactions`)}>Reaction Roles</a>
		<a href={`/app/${id}/starboard`} aria-current={current(`/app/${id}/starboard`)}>Starboard</a>
		<a href={`/app/${id}/commands`} aria-current={current(`/app/${id}/commands`)}>Commands</a>
	</nav>
	<div class="main">
		{#key id}
			{@render children()}
		{/key}
	</div>
</div>
