<script lang="ts">
	import { Input } from 'flowbite-svelte';

	let {
		value = $bindable(),
		min,
		max,
		required = false,
		disabled = false
	}: {
		value?: number;
		min?: number;
		max?: number;
		required?: boolean;
		disabled?: boolean;
	} = $props();
</script>

<Input type="number" {min} {max} {required} {disabled}>
	{#snippet children(props)}
		<input {...props} type="number" bind:value />
	{/snippet}
</Input>
