<script lang="ts">
	import type { Snippet } from 'svelte';
	import { page } from '$app/state';

	let { title, children }: { title: string; children: Snippet } = $props();
	let path = $derived(page.url.pathname);
</script>

<svelte:head><title>{title} - SquirrelBot</title></svelte:head>

<div class="legal">
	<header class="topbar">
		<a class="brand" href="/"><img src="/squirrelbot_brandkit_wide.svg" alt="SquirrelBot" width="240" /></a>
		<nav class="legal-nav" aria-label="Policies">
			<a href="/terms" aria-current={path === '/terms' ? 'page' : undefined}>Terms</a>
			<a href="/privacy" aria-current={path === '/privacy' ? 'page' : undefined}>Privacy</a>
		</nav>
	</header>
	<main class="main legal-body">
		<p class="kicker">SquirrelBot</p>
		<h1>{title}</h1>
		<p class="sub">Effective November 7, 2026. Boxing Octopus publishes SquirrelBot. These pages cover the dashboard and bot at this site.</p>
		{@render children()}
	</main>
	<footer class="legal-foot">
		<p>
			Questions: <a href="https://github.com/BoxingOctopus/squirrelbot">github.com/BoxingOctopus/squirrelbot</a>
		</p>
	</footer>
</div>
