<script lang="ts">
	let { message = '' }: { message?: string } = $props();
</script>

<h1>Something went wrong</h1>
<p>{message || 'This page did not load.'}</p>
<p>Go back to your servers and try that page again.</p>
<p><a href="/app">Back to servers</a></p>
