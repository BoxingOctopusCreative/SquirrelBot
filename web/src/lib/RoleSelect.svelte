<script lang="ts">
	import { Select } from 'flowbite-svelte';
	import type { Role } from '#lib/types';

	let {
		roles,
		value = $bindable(''),
		allowEmpty = true,
		emptyLabel = 'None'
	}: {
		roles: Role[];
		value?: string;
		allowEmpty?: boolean;
		emptyLabel?: string;
	} = $props();

	let items = $derived([
		...(allowEmpty ? [{ value: '', name: emptyLabel }] : []),
		...roles.map((role) => ({ value: role.id, name: role.name, disabled: role.managed }))
	]);
</script>

<Select {items} bind:value placeholder="" />
