<script lang="ts">
	import { Select } from 'flowbite-svelte';
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

	let items = $derived([
		...(allowEmpty ? [{ value: '', name: emptyLabel }] : []),
		...channels.map((channel) => ({ value: channel.id, name: `${prefix}${channel.name}` }))
	]);
</script>

<Select {items} bind:value placeholder="" />
