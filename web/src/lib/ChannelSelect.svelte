<script lang="ts">
	import type { Channel } from '#lib/types';

	let {
		channels,
		value = $bindable(''),
		allowEmpty = true,
		emptyLabel = 'None',
		prefix = '#'
	}: {
		channels: Channel[];
		value?: string;
		allowEmpty?: boolean;
		emptyLabel?: string;
		prefix?: string;
	} = $props();
</script>

<select
	class="block w-full rounded-lg border border-gray-300 bg-gray-50 px-2.5 py-2.5 text-sm text-gray-900 focus:border-primary-500 focus:ring-primary-500 focus:outline-hidden dark:border-gray-600 dark:bg-gray-700 dark:text-white dark:focus:border-primary-500 dark:focus:ring-primary-500"
	value={value}
	onchange={(event) => {
		value = event.currentTarget.value;
	}}
>
	{#if allowEmpty}<option value="">{emptyLabel}</option>{/if}
	{#each channels as channel (channel.id)}
		<option value={channel.id}>{prefix}{channel.name}</option>
	{/each}
</select>
