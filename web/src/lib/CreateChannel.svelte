<script lang="ts">
	import { refreshAll } from '$app/navigation';
	import { sendJSON } from '#lib/api';
	import type { Channel } from '#lib/types';

	let {
		guildId,
		purpose,
		after = '',
		oncreated
	}: {
		guildId: string;
		purpose: string;
		after?: string;
		oncreated?: (channel: Channel) => void;
	} = $props();

	let channelName = $state('');
	let busy = $state(false);
	let notice = $state('');

	async function create() {
		const typed = channelName.trim();
		if (!typed) {
			notice = 'Name the channel.';
			return;
		}
		busy = true;
		notice = '';
		try {
			const channel = await sendJSON<Channel & { created: boolean }>(
				`/api/guilds/${guildId}/channels`,
				'POST',
				{ name: typed, purpose }
			);
			await refreshAll();
			oncreated?.(channel);
			channelName = '';
			notice = channel.created
				? `Created #${channel.name}.${after ? ` ${after}` : ''}`
				: `#${channel.name} is already there.`;
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not create that channel.';
		} finally {
			busy = false;
		}
	}
</script>

<input
	class="channel-name"
	bind:value={channelName}
	maxlength="100"
	placeholder="Channel name"
	disabled={busy}
/>
<button class="quiet" type="button" disabled={busy} onclick={create}>Create channel</button>
{#if notice}<span class="banner">{notice}</span>{/if}
