<script lang="ts">
	let { text = '', ok = false }: { text?: string; ok?: boolean } = $props();

	let positive = $derived(
		ok ||
			/^(Saved\.|Created |Scheduled\.|Sent\.|Filled |Allowlist saved|Granted to |Automod saved|Roles channel saved|Message posted|Template saved)/.test(
				text
			) ||
			text.endsWith('is already there.')
	);
</script>

{#if text}
	<p class={['banner', { ok: positive }]} role="status">{text}</p>
{/if}
