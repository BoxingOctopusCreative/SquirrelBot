<script lang="ts">
	import { refreshAll } from '$app/navigation';
	import { untrack } from 'svelte';
	import { sendJSON, when } from '#lib/api';
	import ChannelSelect from '#lib/ChannelSelect.svelte';
	import Field from '#lib/Field.svelte';
	import type { Bundle, GateMember } from '#lib/types';

	let { data }: { data: { bundle: Bundle; pending: GateMember[]; recent: GateMember[] } } = $props();
	let gate = $state(
		untrack(() => ({
			...data.bundle.gate,
			exemptRoleIds: [...data.bundle.gate.exemptRoleIds]
		}))
	);
	let questions = $state(untrack(() => [...(data.bundle.gate.questions ?? [])]));
	let deadlineHours = $state(untrack(() => Math.max(1, Math.round(data.bundle.gate.deadlineSeconds / 3600))));
	let warnHours = $state(untrack(() => Math.round(data.bundle.gate.warnBeforeSeconds / 3600)));
	let panelChannel = $state(untrack(() => data.bundle.channels[0]?.id ?? ''));
	let notice = $state('');
	let busy = $state(false);

	function addQuestion() {
		if (questions.length >= 4) return;
		questions.push('');
	}

	function removeQuestion(index: number) {
		questions.splice(index, 1);
	}

	function toggleExempt(id: string) {
		gate.exemptRoleIds = gate.exemptRoleIds.includes(id)
			? gate.exemptRoleIds.filter((role) => role !== id)
			: [...gate.exemptRoleIds, id];
	}

	async function save() {
		busy = true;
		notice = '';
		try {
			await sendJSON(`/api/guilds/${data.bundle.guild.id}/gate`, 'PUT', {
				...gate,
				questions,
				deadlineSeconds: deadlineHours * 3600,
				warnBeforeSeconds: warnHours * 3600
			});
			notice = 'Saved.';
			await refreshAll();
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not save.';
		} finally {
			busy = false;
		}
	}

	async function postPanel() {
		notice = '';
		try {
			await sendJSON(`/api/guilds/${data.bundle.guild.id}/gate/panel`, 'POST', { channelId: panelChannel });
			notice = 'Message posted.';
			await refreshAll();
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not post.';
		}
	}
</script>

<svelte:head><title>Server application · SquirrelBot</title></svelte:head>
<p class="kicker">Application</p>
<h1>Server application</h1>
<p class="sub">New members get a deadline to answer the questions. The first question is always "What is your age?". An answer under 18 removes them immediately. Finishing the form clears the deadline.</p>

<form class="panel" onsubmit={(event) => { event.preventDefault(); save(); }}>
	<label class="check"><input type="checkbox" bind:checked={gate.enabled} /> Kick people who miss the deadline</label>
	<Field label="Questions" hint="Age is always first and cannot be changed. Add up to 4 more. Each one can be 45 characters, which is Discord's limit for a form label.">
		<input value="What is your age?" disabled />
		{#each questions as _, index (index)}
			<div class="question-row">
				<input bind:value={questions[index]} maxlength="45" placeholder="Question" required />
				<button class="quiet" type="button" onclick={() => removeQuestion(index)}>Remove</button>
			</div>
		{/each}
		{#if questions.length < 4}
			<button class="quiet" type="button" onclick={addQuestion}>Add question</button>
		{/if}
	</Field>
	<div class="split">
		<Field label="Hours to comply"><input type="number" min="1" max="720" bind:value={deadlineHours} /></Field>
		<Field label="Warn this many hours before" hint="0 skips the warning."><input type="number" min="0" max="719" bind:value={warnHours} /></Field>
	</div>
	<Field label="Kick reason"><input bind:value={gate.kickReason} maxlength="400" required /></Field>
	<Field label="Direct message on join" hint="Placeholders: {'{server} {duration}'}">
		<textarea bind:value={gate.dmMessage}></textarea>
	</Field>
	<Field label="Log channel"><ChannelSelect channels={data.bundle.channels} bind:value={gate.logChannelId} /></Field>
	<fieldset class="field">
		<legend>Exempt roles</legend>
		{#each data.bundle.roles as role (role.id)}
			<label class="check"><input type="checkbox" checked={gate.exemptRoleIds.includes(role.id)} onchange={() => toggleExempt(role.id)} /> {role.name}</label>
		{/each}
	</fieldset>
	<div class="actions">
		<button type="submit" disabled={busy}>Save application</button>
		{#if notice}<span class="banner">{notice}</span>{/if}
	</div>
</form>

<section class="panel">
	<h2>Application message</h2>
	<p class="hint">Save first. Members press Answer questions and fill out the form. Posting in the same channel updates the current message.</p>
	<ChannelSelect channels={data.bundle.channels} bind:value={panelChannel} allowEmpty={false} />
	<button type="button" onclick={postPanel}>Post message</button>
</section>

<section>
	<h2>Waiting</h2>
	<div class="list">
		{#each data.pending as member (member.userId)}
			<article class="item">
				<header><strong>{member.userName || member.userId}</strong><span class="meta">{member.status}</span></header>
				<p class="meta">Joined {when(member.joinedAt)} · deadline {when(member.deadlineAt)}</p>
			</article>
		{:else}
			<p class="hint">Nobody is waiting.</p>
		{/each}
	</div>
	<h2>Removed this week</h2>
	<div class="list">
		{#each data.recent as member (member.userId)}
			<p class="item">{member.userName || member.userId} · {when(member.deadlineAt)}</p>
		{:else}
			<p class="hint">No recent kicks.</p>
		{/each}
	</div>
</section>
