<script lang="ts">
	import { Button, Footer, Navbar, NavBrand, Avatar } from 'flowbite-svelte';
	import type { Snippet } from 'svelte';
	import ThemeToggle from '#lib/ThemeToggle.svelte';
	import type { Me } from '#lib/types';

	let { data, children }: { data: { me: Me }; children: Snippet } = $props();

	async function logout() {
		await fetch('/auth/logout', { method: 'POST', credentials: 'same-origin' });
		location.href = '/';
	}
</script>

<div class="flex h-dvh flex-col overflow-hidden">
	<div class="shrink-0">
		<Navbar fluid class="border-b border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800">
			<NavBrand href="/app">
				<img src="/squirrelbot_brandkit_wide.svg" alt="SquirrelBot" class="-my-2.5 h-[calc(3.75rem+2px)] w-auto" />
			</NavBrand>
			<div class="flex flex-wrap items-center gap-2">
				<Button href="/auth/discord/invite" color="alternative">Add a Server</Button>
				{#if data.me.avatarUrl}<Avatar src={data.me.avatarUrl} alt="" size="sm" />{/if}
				<span class="text-sm text-gray-600 dark:text-gray-300">{data.me.username}</span>
				<ThemeToggle />
				<Button color="alternative" type="button" onclick={logout}>Sign Out</Button>
			</div>
		</Navbar>
	</div>
	<div class="min-h-0 flex-1 overflow-y-auto">
		{@render children()}
	</div>
	<Footer class="shrink-0 items-center rounded-none border-t border-gray-200 bg-transparent px-2 py-2.5 shadow-none sm:px-4 md:px-4 md:py-2.5 dark:border-gray-700">
		<p class="legal-links app-bar w-full">
			<span class="legal-credit">A <a href="https://boxingoctop.us">Boxing Octopus Creative</a> Project</span>
			<span class="legal-sep" aria-hidden="true">|</span>
			<span class="legal-credit">Licensed under the <a href="https://www.mozilla.org/MPL/2.0/">Mozilla Public License 2.0</a></span>
			<span class="legal-sep" aria-hidden="true">|</span>
			<a href="/terms">Terms of Service</a>
			<span class="legal-sep" aria-hidden="true">|</span>
			<a href="/privacy">Privacy Policy</a>
			{#if data.me.version}
				<span class="legal-sep" aria-hidden="true">|</span>
				<span class="legal-credit">Version {data.me.version}</span>
			{/if}
		</p>
	</Footer>
</div>
