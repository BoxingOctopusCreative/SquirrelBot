<script lang="ts">
	import { Button, Card, Checkbox, Input, P, Select } from 'flowbite-svelte';
	import { untrack } from 'svelte';
	import { sendJSON } from '#lib/api';
	import ChannelSelect from '#lib/ChannelSelect.svelte';
	import CreateChannel from '#lib/CreateChannel.svelte';
	import Field from '#lib/Field.svelte';
	import Notice from '#lib/Notice.svelte';
	import NumberInput from '#lib/NumberInput.svelte';
	import PageIntro from '#lib/PageIntro.svelte';
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
	let emojiItems = $derived([
		{ value: '', name: 'Other' },
		{ value: '⭐', name: '⭐' },
		...emojis.map((emoji) => ({ value: `${emoji.name}:${emoji.id}`, name: `:${emoji.name}:` }))
	]);

	async function save() {
		busy = true;
		notice = '';
		try {
			const saved = await sendJSON<Starboard>(`/api/guilds/${data.bundle.guild.id}/starboard`, 'PUT', {
				...$state.snapshot(board),
				threshold: Number(board.threshold)
			});
			board = { ...saved, ignoredChannelIds: [...(saved.ignoredChannelIds ?? [])] };
			notice = 'Saved. Starboard updated.';
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not save.';
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>Starboard · SquirrelBot</title></svelte:head>
<PageIntro kicker="Record" title="Starboard">
	<P>Members react with the emoji you pick. When a message reaches the limit, SquirrelBot posts it in the starboard channel. A member's reaction on their own message does not count unless you allow it. The starboard channel itself is never posted.</P>
</PageIntro>
<Card size="xl" class="flex-col p-5">
	<form class="grid gap-4" onsubmit={(event) => { event.preventDefault(); save(); }}>
		<Checkbox bind:checked={board.enabled}>Starboard on</Checkbox>
		{#if !board.enabled}<P>Turn this on to post starred messages in the starboard channel.</P>{/if}
		<Field label="Emoji" hint="One emoji, such as ⭐ or 🔥. Leave it blank to use ⭐. A server emoji is name:id.">
			<span class="flex items-center gap-2">
				<Input bind:value={board.emoji} maxlength={64} />
				{#if emojiIcon}<img class="h-7 w-7 object-contain" src={emojiIcon} alt="" />{/if}
			</span>
		</Field>
		<Field label="Pick an emoji" hint="A server emoji fills the field above. You can still type a different one.">
			<Select
				placeholder=""
				items={emojiItems}
				value={serverEmoji}
				onchange={(event) => {
					const value = event.currentTarget.value;
					if (value) board.emoji = value;
				}}
			/>
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
			<NumberInput min={1} max={20} bind:value={board.threshold} required />
		</Field>
		<Checkbox bind:checked={board.selfStar}>Count a member's reaction on their own message</Checkbox>
		<Checkbox bind:checked={board.nsfw}>Show images from NSFW channels</Checkbox>
		<Field label="Channels to skip" hint="Messages in these channels are never posted.">
			<Select multiple placeholder="" bind:value={board.ignoredChannelIds}>
				{#each data.bundle.channels as channel (channel.id)}
					<option value={channel.id}>#{channel.name}</option>
				{/each}
			</Select>
		</Field>
		<div class="flex flex-wrap items-center gap-3">
			<Button type="submit" disabled={busy}>Save Starboard</Button>
			<Notice text={notice} />
		</div>
	</form>
</Card>
