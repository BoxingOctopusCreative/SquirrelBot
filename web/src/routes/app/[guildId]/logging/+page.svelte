<script lang="ts">
	import { untrack } from 'svelte';
	import { sendJSON } from '#lib/api';
	import ChannelSelect from '#lib/ChannelSelect.svelte';
	import CreateChannel from '#lib/CreateChannel.svelte';
	import Field from '#lib/Field.svelte';
	import type { Bundle, LogConfig } from '#lib/types';

	let { data }: { data: { bundle: Bundle } } = $props();
	let logging = $state<LogConfig>(untrack(() => ({ ...data.bundle.logging })));
	let notice = $state('');
	let busy = $state(false);

	async function save() {
		busy = true;
		notice = '';
		try {
			await sendJSON(`/api/guilds/${data.bundle.guild.id}/logging`, 'PUT', logging);
			notice = 'Saved.';
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not save.';
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>Logging · SquirrelBot</title></svelte:head>
<p class="kicker">Record</p>
<h1>Logging</h1>
<p class="sub">Message and member events land in this channel. If you leave it empty, joins, leaves, and bans use the mod log channel from Welcome. A channel you create here is hidden from members.</p>
<form class="panel" onsubmit={(event) => { event.preventDefault(); save(); }}>
	<Field label="Log channel"><ChannelSelect channels={data.bundle.channels} bind:value={logging.channelId} /></Field>
	<label class="check"><input type="checkbox" bind:checked={logging.messageDelete} /> Message deletes</label>
	<label class="check"><input type="checkbox" bind:checked={logging.messageEdit} /> Message edits</label>
	<label class="check"><input type="checkbox" bind:checked={logging.memberJoin} /> Joins</label>
	<label class="check"><input type="checkbox" bind:checked={logging.memberLeave} /> Leaves</label>
	<label class="check"><input type="checkbox" bind:checked={logging.memberBan} /> Bans</label>
	<div class="actions">
		<button type="submit" disabled={busy}>Save logging</button>
		<CreateChannel
			guildId={data.bundle.guild.id}
			purpose="logging"
			after="Save logging to use it."
			oncreated={(channel) => {
				logging.channelId = channel.id;
			}}
		/>
		{#if notice}<span class="banner">{notice}</span>{/if}
	</div>
</form>
