<script lang="ts">
	import ErrorCopy from '#lib/ErrorCopy.svelte';
	import ThemeToggle from '#lib/ThemeToggle.svelte';

	let { error }: { error: App.Error } = $props();
</script>

<svelte:head><title>Something went wrong · SquirrelBot</title></svelte:head>

<header class="topbar">
	<a class="brand" href="/"><img src="/squirrelbot_brandkit_wide.svg" alt="SquirrelBot" width="240" /></a>
	<ThemeToggle />
</header>
<main id="main" class="empty" tabindex="-1">
	<ErrorCopy message={error.message} />
</main>
