<script lang="ts">
	import { refreshAll } from '$app/navigation';
	import { untrack } from 'svelte';
	import { sendJSON } from '#lib/api';
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

<input
	class="role-name"
	aria-label="Role name"
	bind:value={roleName}
	maxlength="100"
	placeholder="Role name"
	disabled={busy}
	onkeydown={onKeydown}
/>
<button class="quiet" type="button" disabled={busy} onclick={create}>Create role</button>
{#if notice}<span class="banner">{notice}</span>{/if}
