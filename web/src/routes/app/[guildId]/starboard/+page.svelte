<script lang="ts">
	import { untrack } from 'svelte';
	import { sendJSON } from '#lib/api';
	import ChannelSelect from '#lib/ChannelSelect.svelte';
	import CreateChannel from '#lib/CreateChannel.svelte';
	import Field from '#lib/Field.svelte';
	import type { Bundle, Starboard } from '#lib/types';

	let { data }: { data: { bundle: Bundle } } = $props();

	let board = $state<Starboard>(untrack(() => ({
		...data.bundle.starboard,
		ignoredChannelIds: [...(data.bundle.starboard.ignoredChannelIds ?? [])]
	})));
	let notice = $state('');
	let busy = $state(false);

	async function save() {
		busy = true;
		notice = '';
		try {
			const saved = await sendJSON<Starboard>(`/api/guilds/${data.bundle.guild.id}/starboard`, 'PUT', {
				...board,
				threshold: Number(board.threshold)
			});
			board = { ...saved, ignoredChannelIds: [...(saved.ignoredChannelIds ?? [])] };
			notice = 'Saved.';
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not save.';
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>Starboard · SquirrelBot</title></svelte:head>
<p class="kicker">Record</p>
<h1>Starboard</h1>
<p class="sub">Members react with ⭐. When a message reaches the limit, SquirrelBot posts it in the starboard channel. A member's star on their own message does not count unless you allow it. The starboard channel itself is never starred.</p>
<form class="panel" onsubmit={(event) => { event.preventDefault(); save(); }}>
	<label class="check"><input type="checkbox" bind:checked={board.enabled} /> Starboard on</label>
	<Field label="Starboard channel" hint="A channel you create here stays visible to members.">
		<ChannelSelect channels={data.bundle.channels} bind:value={board.channelId} />
	</Field>
	<CreateChannel
		guildId={data.bundle.guild.id}
		purpose="starboard"
		after="Save the channel to use it."
		oncreated={(channel) => { board.channelId = channel.id; }}
	/>
	<Field label="Stars required" hint="From 1 to 20. The usual choice is 3.">
		<input type="number" min="1" max="20" bind:value={board.threshold} required />
	</Field>
	<label class="check"><input type="checkbox" bind:checked={board.selfStar} /> Count a member's star on their own message</label>
	<label class="check"><input type="checkbox" bind:checked={board.nsfw} /> Show images from NSFW channels</label>
	<Field label="Channels to skip" hint="Messages in these channels are never starred.">
		<select multiple bind:value={board.ignoredChannelIds}>
			{#each data.bundle.channels as channel (channel.id)}
				<option value={channel.id}>#{channel.name}</option>
			{/each}
		</select>
	</Field>
	<div class="actions">
		<button type="submit" disabled={busy}>Save starboard</button>
		{#if notice}<p class="notice">{notice}</p>{/if}
	</div>
</form>
