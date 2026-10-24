<script lang="ts">
	import type { Channel } from '#lib/types';

	let {
		channels,
		value = $bindable(''),
		allowEmpty = true,
		emptyLabel = 'None'
	}: {
		channels: Channel[];
		value?: string;
		allowEmpty?: boolean;
		emptyLabel?: string;
	} = $props();
</script>

<select bind:value>
	{#if allowEmpty}<option value="">{emptyLabel}</option>{/if}
	{#each channels as channel (channel.id)}
		<option value={channel.id}>#{channel.name}</option>
	{/each}
</select>
