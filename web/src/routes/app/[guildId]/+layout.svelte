<script lang="ts">
	import { page } from '$app/state';
	import { Sidebar, SidebarGroup, SidebarItem } from 'flowbite-svelte';
	import type { Snippet } from 'svelte';
	import type { Bundle } from '#lib/types';

	let { data, children }: { data: { bundle: Bundle }; children: Snippet } = $props();

	let id = $derived(data.bundle.guild.id);
	let path = $derived(page.url.pathname.replace(/\/$/, ''));

	const sections = [
		['', 'Overview'],
		['/access', 'Dashboard access'],
		['/profile', 'Profile'],
		['/announcements', 'Announcements'],
		['/escalations', 'Admin Firecall Access'],
		['/verification', 'Server Application'],
		['/cases', 'Cases'],
		['/automod', 'Automod'],
		['/logging', 'Logging'],
		['/welcome', 'Welcome'],
		['/reactions', 'Reaction Roles'],
		['/starboard', 'Starboard'],
		['/confessions', 'Confessions'],
		['/tickets', 'Support tickets'],
		['/commands', 'Commands']
	] as const;

	function href(suffix: string): string {
		return `/app/${id}${suffix}`;
	}
</script>

<div class="flex min-h-[calc(100vh-4.5rem)] flex-col md:flex-row">
	<Sidebar
		alwaysOpen
		position="static"
		backdrop={false}
		activateClickOutside={false}
		ariaLabel="Server menu"
		class="h-auto w-full border-e border-gray-200 md:w-56 dark:border-gray-700"
	>
		<nav aria-label="Server sections">
			<SidebarGroup>
				{#each sections as [suffix, label] (suffix)}
					<SidebarItem href={href(suffix)} {label} spanClass="" active={path === href(suffix)} />
				{/each}
			</SidebarGroup>
		</nav>
	</Sidebar>
	<main id="main" class="min-w-0 max-w-4xl flex-1 px-6 py-6" tabindex="-1">
		{#key id}
			{@render children()}
		{/key}
	</main>
</div>
