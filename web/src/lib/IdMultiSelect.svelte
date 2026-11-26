<script lang="ts">
	let {
		options,
		value = $bindable([])
	}: {
		options: { value: string; name: string; disabled?: boolean }[];
		value?: string[];
	} = $props();

	let selected = $derived(new Set(value));
</script>

<select
	class="block w-full rounded-lg border border-gray-300 bg-gray-50 px-2.5 py-2.5 text-sm text-gray-900 focus:border-primary-500 focus:ring-primary-500 focus:outline-hidden dark:border-gray-600 dark:bg-gray-700 dark:text-white dark:focus:border-primary-500 dark:focus:ring-primary-500"
	multiple
	size={Math.min(8, Math.max(options.length, 3))}
	onchange={(event) => {
		value = [...event.currentTarget.selectedOptions].map((option) => option.value);
	}}
>
	{#each options as option (option.value)}
		<option value={option.value} disabled={option.disabled} selected={selected.has(option.value)}>{option.name}</option>
	{/each}
</select>
