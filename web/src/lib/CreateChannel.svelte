<script lang="ts">
	import { refreshAll } from '$app/navigation';
	import { Button, Input, Label } from 'flowbite-svelte';
	import { sendJSON } from '#lib/api';
	import Notice from '#lib/Notice.svelte';
	import type { Channel } from '#lib/types';

	let {
		guildId,
		purpose,
		after = '',
		label = '',
		kind = 'channel',
		oncreated
	}: {
		guildId: string;
		purpose: string;
		after?: string;
		label?: string;
		kind?: 'channel' | 'category';
		oncreated?: (channel: Channel) => void;
	} = $props();

	let channelName = $state('');
	let busy = $state(false);
	let notice = $state('');
	const inputId = $derived(`channel-${guildId}-${purpose}`);
	const fieldLabel = $derived(label || (kind === 'category' ? 'Category name' : 'Channel name'));

	async function create() {
		const typed = channelName.trim();
		if (!typed) {
			notice = kind === 'category' ? 'Name the category.' : 'Name the channel.';
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
			const createdName = kind === 'category' ? `the ${channel.name} category` : `#${channel.name}`;
			const existingName = kind === 'category' ? `The ${channel.name} category` : `#${channel.name}`;
			notice = channel.created
				? `Created ${createdName}.${after ? ` ${after}` : ''}`
				: `${existingName} is already there.`;
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not create that channel.';
		} finally {
			busy = false;
		}
	}
</script>

<div class="grid gap-1.5">
	<Label for={inputId}>{fieldLabel}</Label>
	<div class="flex flex-wrap items-center gap-2">
		<Input
			id={inputId}
			class="w-48"
			bind:value={channelName}
			maxlength={100}
			placeholder={kind === 'category' ? 'Category name' : 'Channel name'}
			disabled={busy}
		/>
		<Button color="alternative" type="button" disabled={busy} onclick={create}>
			{kind === 'category' ? 'Create category' : 'Create channel'}
		</Button>
	</div>
	<Notice text={notice} />
</div>
