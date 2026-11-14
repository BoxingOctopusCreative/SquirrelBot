<script lang="ts">
	import ErrorCopy from '#lib/ErrorCopy.svelte';

	let { error }: { error: App.Error } = $props();
</script>

<svelte:head><title>Something went wrong · SquirrelBot</title></svelte:head>

<main id="main" class="empty" tabindex="-1">
	<ErrorCopy message={error.message} />
</main>
