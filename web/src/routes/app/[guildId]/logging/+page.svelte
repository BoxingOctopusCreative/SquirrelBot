<script lang="ts">
	import { Button, Card, Checkbox, P } from 'flowbite-svelte';
	import { untrack } from 'svelte';
	import { sendJSON } from '#lib/api';
	import ChannelSelect from '#lib/ChannelSelect.svelte';
	import CreateChannel from '#lib/CreateChannel.svelte';
	import Field from '#lib/Field.svelte';
	import Notice from '#lib/Notice.svelte';
	import PageIntro from '#lib/PageIntro.svelte';
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
			notice = 'Saved. Logging updated.';
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not save.';
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>Logging · SquirrelBot</title></svelte:head>
<PageIntro kicker="Record" title="Logging">
	<P>Message and member events land in this channel. If you leave it empty, joins, leaves, and bans use the mod log channel from Welcome. A channel you create here is hidden from members.</P>
</PageIntro>
<Card size="xl" class="flex-col p-5">
	<form class="grid gap-4" onsubmit={(event) => { event.preventDefault(); save(); }}>
		<Checkbox bind:checked={logging.enabled}>Logging on</Checkbox>
		{#if !logging.enabled}<P>Turn this on to record message and member events.</P>{/if}
		<Field label="Log channel"><ChannelSelect channels={data.bundle.channels} bind:value={logging.channelId} /></Field>
		<Checkbox bind:checked={logging.messageDelete}>Message deletes</Checkbox>
		<Checkbox bind:checked={logging.messageEdit}>Message edits</Checkbox>
		<Checkbox bind:checked={logging.memberJoin}>Joins</Checkbox>
		<Checkbox bind:checked={logging.memberLeave}>Leaves</Checkbox>
		<Checkbox bind:checked={logging.memberBan}>Bans</Checkbox>
		<div class="flex flex-wrap items-center gap-3">
			<Button type="submit" disabled={busy}>Save Logging</Button>
			<CreateChannel
				guildId={data.bundle.guild.id}
				purpose="logging"
				after="Save logging to use it."
				oncreated={(channel) => {
					logging.channelId = channel.id;
				}}
			/>
			<Notice text={notice} />
		</div>
	</form>
</Card>
