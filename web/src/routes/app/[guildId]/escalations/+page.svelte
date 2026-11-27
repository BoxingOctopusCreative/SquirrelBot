<script lang="ts">
	import { Button, Card, Checkbox, Heading, Input, P } from 'flowbite-svelte';
	import { refreshAll } from '$app/navigation';
	import { untrack } from 'svelte';
	import { sendJSON, when } from '#lib/api';
	import ChoiceSelect from '#lib/ChoiceSelect.svelte';
	import Field from '#lib/Field.svelte';
	import MemberPick from '#lib/MemberPick.svelte';
	import Notice from '#lib/Notice.svelte';
	import NumberInput from '#lib/NumberInput.svelte';
	import PageIntro from '#lib/PageIntro.svelte';
	import RoleSelect from '#lib/RoleSelect.svelte';
	import type { Bundle, Escalation, EscalationRole } from '#lib/types';

	let { data }: { data: { bundle: Bundle; escalations: Escalation[] } } = $props();
	let allow = $state<EscalationRole[]>(untrack(() => data.bundle.escalationRoles.map((role) => ({ ...role }))));
	let firecallOn = $state(untrack(() => data.bundle.guild.firecallEnabled));
	let roleId = $state('');
	let tier = $state('mod');
	let hours = $state(24);
	let userId = $state('');
	let userName = $state('');
	let grantRole = $state(untrack(() => data.bundle.escalationRoles[0]?.roleId ?? ''));
	let duration = $state('4h');
	let reason = $state('');
	let notice = $state('');
	let busy = $state(false);

	const tiers = [
		{ value: 'mod', name: 'Mod' },
		{ value: 'admin', name: 'Admin' }
	];
	let grantRoles = $derived(allow.map((role) => ({
		id: role.roleId,
		name: roleName(role.roleId),
		position: 0,
		managed: false
	})));

	function addRole() {
		if (!roleId) return;
		allow = [...allow.filter((role) => role.roleId !== roleId), {
			guildId: data.bundle.guild.id,
			roleId,
			tier,
			maxDurationSeconds: hours * 3600
		}];
	}

	async function saveAllow() {
		busy = true;
		notice = '';
		try {
			await sendJSON(`/api/guilds/${data.bundle.guild.id}/escalation-roles`, 'PUT', { roles: allow, enabled: firecallOn });
			notice = 'Allowlist saved.';
			await refreshAll();
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not save.';
		} finally {
			busy = false;
		}
	}

	async function grant() {
		if (!userId) {
			notice = 'Choose a member before granting a role.';
			return;
		}
		busy = true;
		notice = '';
		try {
			await sendJSON(`/api/guilds/${data.bundle.guild.id}/escalations`, 'POST', {
				userId, roleId: grantRole, duration, reason
			});
			notice = `Granted to ${userName || userId}.`;
			reason = '';
			await refreshAll();
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not grant.';
		} finally {
			busy = false;
		}
	}

	async function revoke(id: number) {
		notice = '';
		try {
			await sendJSON(`/api/guilds/${data.bundle.guild.id}/escalations/${id}/revoke`, 'POST', { reason: 'Ended from the dashboard' });
			await refreshAll();
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not revoke.';
		}
	}

	function roleName(id: string): string {
		return data.bundle.roles.find((role) => role.id === id)?.name ?? id;
	}
</script>

<svelte:head><title>Admin Firecall Access · SquirrelBot</title></svelte:head>
<PageIntro kicker="Staff" title="Admin Firecall Access">
	<P>This feature allows admins to grant temporary privilege escalation from one role to another for a given block of time.</P>
</PageIntro>

<section>
<Card size="xl" class="mb-6 flex-col gap-4 p-5">
	<Heading tag="h2" class="font-display text-xl font-semibold">Allowlist</Heading>
	<Checkbox bind:checked={firecallOn}>Admin firecall on</Checkbox>
	{#if !firecallOn}<P>Turn this on before staff can grant a temporary role. Grants that are already running still end on time.</P>{/if}
	<div class="grid gap-4 md:grid-cols-2">
		<Field label="Role"><RoleSelect roles={data.bundle.roles} bind:value={roleId} allowEmpty={false} /></Field>
		<Field label="Tier">
			<ChoiceSelect options={tiers} bind:value={tier} />
		</Field>
	</div>
	<Field label="Longest grant, in hours" hint="From 1 hour to 720 hours (30 days).">
		<NumberInput min={1} max={720} bind:value={hours} />
	</Field>
	<div class="flex flex-wrap items-center gap-3">
		<Button color="alternative" type="button" onclick={addRole}>Add Role</Button>
		<Button type="button" disabled={busy} onclick={saveAllow}>Save Allowlist</Button>
	</div>
	<div class="list grid gap-3">
		{#each allow as role (role.roleId)}
			<Card size="xl" class="flex-col gap-2 p-4">
				<header class="flex items-baseline justify-between gap-3">
					<strong>{roleName(role.roleId)}</strong>
					<span class="font-mono text-sm text-gray-500">{role.tier} · {Math.round(role.maxDurationSeconds / 3600)}h max</span>
				</header>
				<Button color="alternative" size="sm" type="button" onclick={() => (allow = allow.filter((item) => item.roleId !== role.roleId))}>Remove</Button>
			</Card>
		{:else}
			<P class="text-sm">No roles yet. A grant from Discord will fail until you add one.</P>
		{/each}
	</div>
</Card>
</section>

<Card size="xl" class="mb-6 flex-col p-5">
	<form class="grid gap-4" onsubmit={(event) => { event.preventDefault(); grant(); }}>
		<Heading tag="h2" class="font-display text-xl font-semibold">Grant</Heading>
		<MemberPick guildId={data.bundle.guild.id} bind:userId bind:userName />
		<Field label="Role">
			<RoleSelect roles={grantRoles} bind:value={grantRole} allowEmpty={grantRoles.length === 0} />
		</Field>
		<Field label="Duration" hint="Examples: 4h, 2d, 1w."><Input bind:value={duration} required /></Field>
		<Field label="Reason"><Input bind:value={reason} required maxlength={400} /></Field>
		<div class="flex flex-wrap items-center gap-3">
			<Button type="submit" disabled={busy}>Grant Role</Button>
			<Notice text={notice} />
		</div>
	</form>
</Card>

<section class="list grid gap-3">
	{#each data.escalations as item (item.id)}
		<Card size="xl" class="flex-col gap-2 p-4">
			<header class="flex items-baseline justify-between gap-3">
				<strong>{item.userName || item.userId}</strong>
				<span class="font-mono text-sm text-gray-500">{item.status} · {item.tier}</span>
			</header>
			<P>{roleName(item.roleId)} until {when(item.expiresAt)}</P>
			<P class="font-mono text-sm">{item.reason} · {item.grantedByName}</P>
			{#if item.status === 'active'}
				<Button color="red" size="sm" type="button" onclick={() => revoke(item.id)}>End Now</Button>
			{/if}
		</Card>
	{:else}
		<P class="text-sm">No grants yet.</P>
	{/each}
</section>
