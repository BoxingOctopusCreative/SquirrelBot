<script lang="ts">
	import { Helper, Label } from 'flowbite-svelte';
	import type { Snippet } from 'svelte';

	let { label, hint = '', children }: { label: string; hint?: string; children: Snippet } = $props();
</script>

<div class="grid gap-1.5">
	<Label class="flex flex-col gap-1.5">
		<span>{label}</span>
		{@render children()}
	</Label>
	{#if hint}<Helper class="mt-0">{hint}</Helper>{/if}
</div>
