<script lang="ts">
	import { Button, Footer, FooterLink, FooterLinkGroup, Navbar, NavBrand, Avatar } from 'flowbite-svelte';
	import type { Snippet } from 'svelte';
	import ThemeToggle from '#lib/ThemeToggle.svelte';
	import type { Me } from '#lib/types';

	let { data, children }: { data: { me: Me }; children: Snippet } = $props();

	async function logout() {
		await fetch('/auth/logout', { method: 'POST', credentials: 'same-origin' });
		location.href = '/';
	}
</script>

<Navbar fluid class="border-b border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800">
	<NavBrand href="/app">
		<img src="/squirrelbot_brandkit_wide.svg" alt="SquirrelBot" width="200" class="h-8 w-auto" />
	</NavBrand>
	<div class="flex flex-wrap items-center gap-2">
		<Button href="/auth/discord/invite" color="alternative">Add a server</Button>
		{#if data.me.avatarUrl}<Avatar src={data.me.avatarUrl} alt="" size="sm" />{/if}
		<span class="text-sm text-gray-600 dark:text-gray-300">{data.me.username}</span>
		<ThemeToggle />
		<Button color="alternative" type="button" onclick={logout}>Sign out</Button>
	</div>
</Navbar>
{@render children()}
<Footer class="rounded-none bg-transparent p-4 shadow-none">
	<FooterLinkGroup class="flex gap-4">
		<FooterLink href="/terms">Terms</FooterLink>
		<FooterLink href="/privacy">Privacy</FooterLink>
	</FooterLinkGroup>
</Footer>
