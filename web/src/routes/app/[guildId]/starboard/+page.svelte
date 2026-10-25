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
		emoji: data.bundle.starboard.emoji || '⭐',
		ignoredChannelIds: [...(data.bundle.starboard.ignoredChannelIds ?? [])]
	})));
	let notice = $state('');
	let busy = $state(false);
	let emojis = $derived(data.bundle.emojis ?? []);
	let serverEmoji = $derived.by(() => {
		if (board.emoji === '⭐') return '⭐';
		const match = emojis.find((emoji) => `${emoji.name}:${emoji.id}` === board.emoji);
		return match ? `${match.name}:${match.id}` : '';
	});
	let emojiIcon = $derived.by(() => {
		const [name, id] = board.emoji.split(':');
		if (name && /^\d{17,20}$/.test(id ?? '')) {
			return `https://cdn.discordapp.com/emojis/${id}.png`;
		}
		return '';
	});

	async function save() {
		busy = true;
		notice = '';
		try {
			const saved = await sendJSON<Starboard>(`/api/guilds/${data.bundle.guild.id}/starboard`, 'PUT', {
				...$state.snapshot(board),
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
<p class="sub">Members react with the emoji you pick. When a message reaches the limit, SquirrelBot posts it in the starboard channel. A member's reaction on their own message does not count unless you allow it. The starboard channel itself is never posted.</p>
<form class="panel" onsubmit={(event) => { event.preventDefault(); save(); }}>
	<label class="check"><input type="checkbox" bind:checked={board.enabled} /> Starboard on</label>
	{#if !board.enabled}<p class="sub">Turn this on to post starred messages in the starboard channel.</p>{/if}
	<Field label="Emoji" hint="One emoji, such as ⭐ or 🔥. Leave it blank to use ⭐. A server emoji is name:id.">
		<span class="emoji-pick">
			<input bind:value={board.emoji} maxlength="64" />
			{#if emojiIcon}<img src={emojiIcon} alt="" />{/if}
		</span>
	</Field>
	<Field label="Pick an emoji" hint="A server emoji fills the field above. You can still type a different one.">
		<select
			value={serverEmoji}
			onchange={(event) => {
				const value = event.currentTarget.value;
				if (value) board.emoji = value;
			}}
		>
			<option value="">Other</option>
			<option value="⭐">⭐</option>
			{#each emojis as emoji (emoji.id)}
				<option value={`${emoji.name}:${emoji.id}`}>:{emoji.name}:</option>
			{/each}
		</select>
	</Field>
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
	<label class="check"><input type="checkbox" bind:checked={board.selfStar} /> Count a member's reaction on their own message</label>
	<label class="check"><input type="checkbox" bind:checked={board.nsfw} /> Show images from NSFW channels</label>
	<Field label="Channels to skip" hint="Messages in these channels are never posted.">
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
