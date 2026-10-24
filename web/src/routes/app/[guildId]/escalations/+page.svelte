<script lang="ts">
	import { refreshAll } from '$app/navigation';
	import { untrack } from 'svelte';
	import { sendJSON, when } from '#lib/api';
	import Field from '#lib/Field.svelte';
	import MemberPick from '#lib/MemberPick.svelte';
	import RoleSelect from '#lib/RoleSelect.svelte';
	import type { Bundle, Escalation, EscalationRole } from '#lib/types';

	let { data }: { data: { bundle: Bundle; escalations: Escalation[] } } = $props();
	let allow = $state<EscalationRole[]>(untrack(() => data.bundle.escalationRoles.map((role) => ({ ...role }))));
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
			await sendJSON(`/api/guilds/${data.bundle.guild.id}/escalation-roles`, 'PUT', { roles: allow });
			notice = 'Allowlist saved.';
			await refreshAll();
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not save.';
		} finally {
			busy = false;
		}
	}

	async function grant() {
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
<p class="kicker">Staff</p>
<h1>Admin Firecall Access</h1>
<p class="sub">This feature allows admins to grant temporary privilege escalation from one role to another for a given block of time.</p>

<section class="panel">
	<h2>Allowlist</h2>
	<div class="split">
		<Field label="Role"><RoleSelect roles={data.bundle.roles} bind:value={roleId} allowEmpty={false} /></Field>
		<Field label="Tier">
			<select bind:value={tier}><option value="mod">Mod</option><option value="admin">Admin</option></select>
		</Field>
	</div>
	<Field label="Longest grant, in hours" hint="From 1 hour to 720 hours (30 days).">
		<input type="number" min="1" max="720" bind:value={hours} />
	</Field>
	<div class="actions">
		<button class="quiet" type="button" onclick={addRole}>Add role</button>
		<button type="button" disabled={busy} onclick={saveAllow}>Save allowlist</button>
	</div>
	<div class="list">
		{#each allow as role (role.roleId)}
			<div class="item">
				<header>
					<strong>{roleName(role.roleId)}</strong>
					<span class="meta">{role.tier} · {Math.round(role.maxDurationSeconds / 3600)}h max</span>
				</header>
				<button class="quiet" type="button" onclick={() => (allow = allow.filter((item) => item.roleId !== role.roleId))}>Remove</button>
			</div>
		{:else}
			<p class="hint">No roles yet. A grant from Discord will fail until you add one.</p>
		{/each}
	</div>
</section>

<form class="panel" onsubmit={(event) => { event.preventDefault(); grant(); }}>
	<h2>Grant</h2>
	<MemberPick guildId={data.bundle.guild.id} bind:userId bind:userName />
	<Field label="Role">
		<select bind:value={grantRole}>
			{#each allow as role (role.roleId)}
				<option value={role.roleId}>{roleName(role.roleId)}</option>
			{/each}
		</select>
	</Field>
	<Field label="Duration" hint="Examples: 4h, 2d, 1w."><input bind:value={duration} required /></Field>
	<Field label="Reason"><input bind:value={reason} required maxlength="400" /></Field>
	<div class="actions">
		<button type="submit" disabled={busy || !userId}>Grant role</button>
		{#if notice}<span class="banner">{notice}</span>{/if}
	</div>
</form>

<section class="list">
	{#each data.escalations as item (item.id)}
		<article class="item">
			<header>
				<strong>{item.userName || item.userId}</strong>
				<span class="meta">{item.status} · {item.tier}</span>
			</header>
			<p>{roleName(item.roleId)} until {when(item.expiresAt)}</p>
			<p class="meta">{item.reason} · {item.grantedByName}</p>
			{#if item.status === 'active'}
				<button class="danger" type="button" onclick={() => revoke(item.id)}>End now</button>
			{/if}
		</article>
	{:else}
		<p class="hint">No grants yet.</p>
	{/each}
</section>
