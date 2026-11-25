<script lang="ts">
	import { refreshAll } from '$app/navigation';
	import { Button, Input, Label } from 'flowbite-svelte';
	import { untrack } from 'svelte';
	import { sendJSON } from '#lib/api';
	import Notice from '#lib/Notice.svelte';
	import type { Role } from '#lib/types';

	let {
		guildId,
		defaultName = '',
		oncreated
	}: {
		guildId: string;
		defaultName?: string;
		oncreated?: (role: Role) => void | Promise<void>;
	} = $props();

	let roleName = $state(untrack(() => defaultName));
	let busy = $state(false);
	let notice = $state('');
	const inputId = $derived(`role-${guildId}`);

	async function create() {
		const typed = roleName.trim();
		if (!typed) {
			notice = 'Name the role.';
			return;
		}
		busy = true;
		notice = '';
		try {
			const role = await sendJSON<Role & { created: boolean }>(`/api/guilds/${guildId}/roles`, 'POST', {
				name: typed
			});
			await refreshAll();
			await oncreated?.(role);
			roleName = defaultName;
			notice = role.created ? `Created ${role.name}.` : `${role.name} is already there.`;
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not create that role.';
		} finally {
			busy = false;
		}
	}

	function onKeydown(event: KeyboardEvent) {
		if (event.key !== 'Enter') return;
		event.preventDefault();
		create();
	}
</script>

<div class="grid gap-1.5">
	<Label for={inputId}>Role name</Label>
	<div class="flex flex-wrap items-center gap-2">
		<Input
			id={inputId}
			class="w-48"
			bind:value={roleName}
			maxlength={100}
			placeholder="Role name"
			disabled={busy}
			onkeydown={onKeydown}
		/>
		<Button color="alternative" type="button" disabled={busy} onclick={create}>Create Role</Button>
	</div>
	<Notice text={notice} />
</div>
