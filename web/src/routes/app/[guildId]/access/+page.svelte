<script lang="ts">
	import { Button, Card, P } from 'flowbite-svelte';
	import { refreshAll } from '$app/navigation';
	import { untrack } from 'svelte';
	import { sendJSON } from '#lib/api';
	import CreateRole from '#lib/CreateRole.svelte';
	import Field from '#lib/Field.svelte';
	import Notice from '#lib/Notice.svelte';
	import PageIntro from '#lib/PageIntro.svelte';
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
<PageIntro kicker="Who can open this" title="Dashboard access">
	<P>
		People with Manage Server can open this dashboard. A role saved here can open it and change settings. Only Manage
		Server can change this list. A role SquirrelBot creates has no Discord permissions of its own. Assign it in Discord
		to the people who should sign in.
	</P>
</PageIntro>

{#if data.bundle.manageServer}
	<Card size="xl" class="flex-col gap-4 p-5">
		<CreateRole guildId={data.bundle.guild.id} defaultName="Moderator" oncreated={grantCreated} />
		<P class="text-sm">Or add a role that already exists.</P>
		<Field label="Role">
			<RoleSelect roles={data.bundle.roles} bind:value={roleId} allowEmpty={false} />
		</Field>
		<div class="flex flex-wrap items-center gap-3">
			<Button color="alternative" type="button" onclick={addRole}>Add role</Button>
			<Button type="button" disabled={busy} onclick={save}>Save access</Button>
			<Notice text={notice} />
		</div>
		<div class="list grid gap-3">
			{#each roleIds as id (id)}
				<Card size="xl" class="flex-row items-center justify-between gap-3 p-4">
					<strong>{roleName(id)}</strong>
					<Button color="alternative" size="sm" type="button" onclick={() => (roleIds = roleIds.filter((item) => item !== id))}>Remove</Button>
				</Card>
			{:else}
				<P class="text-sm">No extra roles yet. Moderators stay signed out until you add one.</P>
			{/each}
		</div>
	</Card>
{:else}
	<section class="list grid gap-3">
		{#each data.bundle.managerRoleIds as id (id)}
			<Card size="xl" class="p-4"><strong>{roleName(id)}</strong></Card>
		{:else}
			<P class="text-sm">No extra roles yet.</P>
		{/each}
	</section>
	<P class="mt-4 text-sm">You can change settings because you have one of these roles. Someone with Manage Server edits the list.</P>
{/if}
