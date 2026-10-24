<script lang="ts">
	let { error }: { error: App.Error } = $props();
</script>

<main class="empty">
	<h1>Something went wrong</h1>
	<p>{error.message}</p>
	<p><a href="/app">Back to servers</a></p>
</main>
