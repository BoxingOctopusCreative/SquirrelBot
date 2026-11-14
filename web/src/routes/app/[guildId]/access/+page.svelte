<script lang="ts">
	import { refreshAll } from '$app/navigation';
	import { untrack } from 'svelte';
	import { sendJSON } from '#lib/api';
	import CreateRole from '#lib/CreateRole.svelte';
	import Field from '#lib/Field.svelte';
	import Notice from '#lib/Notice.svelte';
	import RoleSelect from '#lib/RoleSelect.svelte';
	import type { Bundle, Role } from '#lib/types';

	let { data }: { data: { bundle: Bundle } } = $props();
	let roleIds = $state<string[]>(untrack(() => [...data.bundle.managerRoleIds]));
	let roleId = $state('');
	let notice = $state('');
	let busy = $state(false);

	function addRole() {
		if (!roleId || roleIds.includes(roleId)) return;
		roleIds = [...roleIds, roleId];
	}

	async function grantCreated(role: Role) {
		if (!roleIds.includes(role.id)) roleIds = [...roleIds, role.id];
		await save();
	}

	async function save() {
		busy = true;
		notice = '';
		try {
			const saved = await sendJSON<{ roleIds: string[] }>(
				`/api/guilds/${data.bundle.guild.id}/manager-roles`,
				'PUT',
				{ roleIds }
			);
			roleIds = saved.roleIds;
			notice = 'Saved. Dashboard access updated.';
			await refreshAll();
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not save.';
		} finally {
			busy = false;
		}
	}

	function roleName(id: string): string {
		return data.bundle.roles.find((role) => role.id === id)?.name ?? id;
	}
</script>

<svelte:head><title>Dashboard access · SquirrelBot</title></svelte:head>
<p class="kicker">Who can open this</p>
<h1>Dashboard access</h1>
<p class="sub">
	People with Manage Server can open this dashboard. A role saved here can open it and change settings. Only Manage
	Server can change this list. A role SquirrelBot creates has no Discord permissions of its own. Assign it in Discord
	to the people who should sign in.
</p>

{#if data.bundle.manageServer}
	<section class="panel">
		<div class="actions">
			<CreateRole guildId={data.bundle.guild.id} defaultName="Moderator" oncreated={grantCreated} />
		</div>
		<p class="hint">Or add a role that already exists.</p>
		<Field label="Role">
			<RoleSelect roles={data.bundle.roles} bind:value={roleId} allowEmpty={false} />
		</Field>
		<div class="actions">
			<button class="quiet" type="button" onclick={addRole}>Add role</button>
			<button type="button" disabled={busy} onclick={save}>Save access</button>
			<Notice text={notice} />
		</div>
		<div class="list">
			{#each roleIds as id (id)}
				<div class="item">
					<header>
						<strong>{roleName(id)}</strong>
						<button class="quiet" type="button" onclick={() => (roleIds = roleIds.filter((item) => item !== id))}>Remove</button>
					</header>
				</div>
			{:else}
				<p class="hint">No extra roles yet. Moderators stay signed out until you add one.</p>
			{/each}
		</div>
	</section>
{:else}
	<section class="list">
		{#each data.bundle.managerRoleIds as id (id)}
			<div class="item"><strong>{roleName(id)}</strong></div>
		{:else}
			<p class="hint">No extra roles yet.</p>
		{/each}
	</section>
	<p class="hint">You can change settings because you have one of these roles. Someone with Manage Server edits the list.</p>
{/if}
