<script lang="ts">
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
</script>

<select bind:value>
	{#if allowEmpty}<option value="">{emptyLabel}</option>{/if}
	{#each roles as role (role.id)}
		<option value={role.id} disabled={role.managed}>{role.name}</option>
	{/each}
</select>
