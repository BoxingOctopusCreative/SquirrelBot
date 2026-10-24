<script lang="ts">
	import { refreshAll } from '$app/navigation';
	import { sendJSON } from '#lib/api';
	import Field from '#lib/Field.svelte';
	import type { Bundle, CustomCommand } from '#lib/types';

	let { data }: { data: { bundle: Bundle; commands: CustomCommand[] } } = $props();
	let name = $state('');
	let response = $state('');
	let notice = $state('');

	async function save(event: SubmitEvent) {
		event.preventDefault();
		notice = '';
		try {
			await sendJSON(`/api/guilds/${data.bundle.guild.id}/commands`, 'POST', { name, response });
			name = '';
			response = '';
			notice = 'Saved.';
			await refreshAll();
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not save.';
		}
	}

	async function remove(command: string) {
		notice = '';
		try {
			await sendJSON(`/api/guilds/${data.bundle.guild.id}/commands/${command}`, 'DELETE');
			await refreshAll();
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not delete.';
		}
	}
</script>

<svelte:head><title>Commands · SquirrelBot</title></svelte:head>
<p class="kicker">Shortcuts</p>
<h1>Commands</h1>
<p class="sub">Members type {data.bundle.guild.prefix}name. Mentions in the response are not pinged.</p>
<form class="panel" onsubmit={save}>
	<Field label="Name"><input bind:value={name} required maxlength="32" pattern="[a-z0-9_-]+" /></Field>
	<Field label="Response"><textarea bind:value={response} maxlength="1500" required></textarea></Field>
	<div class="actions">
		<button type="submit">Save command</button>
		{#if notice}<span class="banner">{notice}</span>{/if}
	</div>
</form>
<section class="list">
	{#each data.commands as command (command.name)}
		<article class="item">
			<header><strong>{data.bundle.guild.prefix}{command.name}</strong></header>
			<p>{command.response}</p>
			<button class="quiet" type="button" onclick={() => remove(command.name)}>Delete</button>
		</article>
	{:else}
		<p class="hint">No custom commands yet.</p>
	{/each}
</section>
