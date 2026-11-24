<script lang="ts">
	import { Heading } from 'flowbite-svelte';
	import type { Snippet } from 'svelte';

	let { kicker = '', title, children }: { kicker?: string; title: string; children?: Snippet } = $props();
</script>

{#if kicker}
	<p class="mb-1 text-xs font-semibold tracking-widest text-primary-600 uppercase">{kicker}</p>
{/if}
<Heading tag="h1" class="mb-2 font-display text-3xl font-semibold">{title}</Heading>
{#if children}
	<div class="mb-6 max-w-3xl text-gray-600 dark:text-gray-300">
		{@render children()}
	</div>
{/if}
