<script lang="ts">
	import { page } from '$app/state';
	import { Sidebar, SidebarGroup, SidebarItem } from 'flowbite-svelte';
	import type { Snippet } from 'svelte';
	import type { Bundle } from '#lib/types';

	let { data, children }: { data: { bundle: Bundle }; children: Snippet } = $props();

	let id = $derived(data.bundle.guild.id);
	let path = $derived(page.url.pathname.replace(/\/$/, ''));

	const groups = [
		{
			title: 'Server',
			items: [
				['', 'Overview'],
				['/profile', 'Profile'],
				['/access', 'Dashboard Access']
			]
		},
		{
			title: 'Moderation',
			items: [
				['/cases', 'Moderation Log'],
				['/automod', 'Automod'],
				['/purges', 'Timed Channel Purge'],
				['/logging', 'Logging'],
				['/escalations', 'Admin Firecall Access']
			]
		},
		{
			title: 'Members',
			items: [
				['/welcome', 'Welcome'],
				['/verification', 'Server Application']
			]
		},
		{
			title: 'Community',
			items: [
				['/announcements', 'Announcements'],
				['/reactions', 'Reaction Roles'],
				['/starboard', 'Starboard'],
				['/levels', 'Levels'],
				['/confessions', 'Confessions'],
				['/tickets', 'Support Tickets'],
				['/commands', 'Commands']
			]
		}
	] as const;

	function href(suffix: string): string {
		return `/app/${id}${suffix}`;
	}
</script>

<div class="flex min-h-full flex-col md:flex-row">
	<Sidebar
		alwaysOpen
		position="static"
		backdrop={false}
		activateClickOutside={false}
		ariaLabel="Server menu"
		class="h-auto w-full border-e border-gray-200 bg-[#f3f4f6] md:w-56 dark:border-gray-700 dark:bg-[#1a2331]"
		classes={{ div: 'bg-[#f3f4f6] dark:bg-[#1a2331]' }}
	>
		<nav aria-label="Server sections" class="flex flex-col">
			{#each groups as group, index (group.title)}
				<div class={index > 0 ? 'mt-4 border-t border-gray-200 pt-4 dark:border-gray-700' : ''}>
					<p class="px-2 pb-1 text-xs font-semibold tracking-wide text-gray-500 dark:text-gray-400">{group.title}</p>
					<SidebarGroup class="space-y-0">
						{#each group.items as [suffix, label] (suffix)}
							<SidebarItem href={href(suffix)} {label} spanClass="" aClass="px-2! py-0!" active={path === href(suffix)} />
						{/each}
					</SidebarGroup>
				</div>
			{/each}
		</nav>
	</Sidebar>
	<main id="main" class="min-w-0 max-w-4xl flex-1 px-6 py-6" tabindex="-1">
		{#key id}
			{@render children()}
		{/key}
	</main>
</div>
