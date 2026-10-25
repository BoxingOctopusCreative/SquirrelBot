<script lang="ts">
	import type { Snippet } from 'svelte';
	import Mark from '#lib/Mark.svelte';
	import type { Me } from '#lib/types';

	let { data, children }: { data: { me: Me }; children: Snippet } = $props();

	async function logout() {
		await fetch('/auth/logout', { method: 'POST', credentials: 'same-origin' });
		location.href = '/';
	}
</script>

<header class="topbar">
	<a class="brand" href="/app"><img src="/squirrelbot_brandkit_wide.svg" alt="SquirrelBot" width="240" /></a>
	<div class="who">
		<a class="button quiet" href="/auth/discord">Add a server</a>
		{#if data.me.avatarUrl}<img class="avatar" src={data.me.avatarUrl} alt="" />{/if}
		<span>{data.me.username}</span>
		<button class="quiet" type="button" onclick={logout}>Sign out</button>
	</div>
</header>
{@render children()}
