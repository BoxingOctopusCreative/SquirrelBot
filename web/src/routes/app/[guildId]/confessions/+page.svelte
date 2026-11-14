<script lang="ts">
	import { untrack } from 'svelte';
	import { sendJSON } from '#lib/api';
	import ChannelSelect from '#lib/ChannelSelect.svelte';
	import CreateChannel from '#lib/CreateChannel.svelte';
	import Field from '#lib/Field.svelte';
	import Notice from '#lib/Notice.svelte';
	import type { Bundle, ConfessionConfig } from '#lib/types';

	let { data }: { data: { bundle: Bundle } } = $props();

	let cfg = $state<ConfessionConfig>(untrack(() => ({ ...data.bundle.confessions })));
	let notice = $state('');
	let busy = $state(false);

	async function save() {
		busy = true;
		notice = '';
		try {
			const saved = await sendJSON<ConfessionConfig>(
				`/api/guilds/${data.bundle.guild.id}/confessions`,
				'PUT',
				$state.snapshot(cfg)
			);
			cfg = { ...saved };
			notice = 'Saved. Confessions updated.';
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not save.';
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>Confessions · SquirrelBot</title></svelte:head>
<p class="kicker">Community</p>
<h1>Confessions</h1>
<p class="sub">
	Members send a note with /confession, or with Submit a Confession on an approved post. That button opens a form for the text and an optional file. The note lands in the review channel and tags the mod role. Moderators approve or reject it there. An approved note is posted in the confessions channel, and the author's name stays off that post. Set both channels here, or in Discord with /confession-review and /confession-channel. Those two commands are limited to moderators.
</p>
<form class="panel" onsubmit={(event) => { event.preventDefault(); save(); }}>
	<Field label="Review channel" hint="Each confession tags the mod role from Welcome. A channel you create here is hidden from members. The mod role and admin role from Welcome can read it.">
		<ChannelSelect channels={data.bundle.channels} bind:value={cfg.reviewChannelId} />
	</Field>
	<CreateChannel
		guildId={data.bundle.guild.id}
		purpose="confession-review"
		after="Save confessions to use it."
		oncreated={(channel) => { cfg.reviewChannelId = channel.id; }}
	/>
	<Field label="Confessions channel" hint="Approved confessions are posted here. A channel you create here stays visible to members.">
		<ChannelSelect channels={data.bundle.channels} bind:value={cfg.channelId} />
	</Field>
	<CreateChannel
		guildId={data.bundle.guild.id}
		purpose="confessions"
		after="Save confessions to use it."
		oncreated={(channel) => { cfg.channelId = channel.id; }}
	/>
	<div class="actions">
		<button type="submit" disabled={busy}>Save confessions</button>
		<Notice text={notice} />
	</div>
</form>
