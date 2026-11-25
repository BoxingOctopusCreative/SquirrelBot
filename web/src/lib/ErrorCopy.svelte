<script lang="ts">
	import { Button, Heading, P } from 'flowbite-svelte';

	let { message = '' }: { message?: string } = $props();
</script>

<Heading tag="h1" class="mb-3 font-display text-3xl font-semibold">Something went wrong</Heading>
<P>{message || 'This page did not load.'}</P>
<P class="mt-2">Go back to your servers and try that page again.</P>
<Button href="/app" class="mt-4">Back to Servers</Button>
