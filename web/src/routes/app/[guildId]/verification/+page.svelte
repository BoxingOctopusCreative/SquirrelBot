<script lang="ts">
	import { Button, Card, Checkbox, Heading, Helper, Input, Label, P, Textarea } from 'flowbite-svelte';
	import { refreshAll } from '$app/navigation';
	import { untrack } from 'svelte';
	import { sendJSON, when } from '#lib/api';
	import ChannelSelect from '#lib/ChannelSelect.svelte';
	import Field from '#lib/Field.svelte';
	import Notice from '#lib/Notice.svelte';
	import NumberInput from '#lib/NumberInput.svelte';
	import PageIntro from '#lib/PageIntro.svelte';
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
			notice = 'Saved. Application updated.';
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

<svelte:head><title>Server Application · SquirrelBot</title></svelte:head>
<PageIntro kicker="Application" title="Server Application">
	<P>New members get a deadline to answer the questions. The first question is always "What is your age?". An answer under 18 removes them immediately. Finishing the form clears the deadline.</P>
</PageIntro>

<Card size="xl" class="mb-6 flex-col p-5">
	<form class="grid gap-4" onsubmit={(event) => { event.preventDefault(); save(); }}>
		<Checkbox bind:checked={gate.enabled}>Kick people who miss the deadline</Checkbox>
		<fieldset class="grid gap-3">
			<legend class="text-sm font-medium">Questions</legend>
			<Helper>Age is always first and cannot be changed. Add up to 4 more. Each one can be 45 characters, which is Discord's limit for a form label.</Helper>
			<div class="flex flex-wrap items-center gap-2">
				<Label for="age-question" class="w-32">Age question</Label>
				<Input id="age-question" class="min-w-48 flex-1" value="What is your age?" disabled />
			</div>
			{#each questions as _, index (index)}
				<div class="flex flex-wrap items-center gap-2">
					<Label for="question-{index}" class="w-32">Question {index + 2}</Label>
					<Input id="question-{index}" class="min-w-48 flex-1" bind:value={questions[index]} maxlength={45} placeholder="Question" required />
					<Button color="alternative" type="button" onclick={() => removeQuestion(index)}>Remove</Button>
				</div>
			{/each}
			{#if questions.length < 4}
				<Button color="alternative" type="button" onclick={addQuestion}>Add Question</Button>
			{/if}
		</fieldset>
		<div class="grid gap-4 md:grid-cols-2">
			<Field label="Hours to comply"><NumberInput min={1} max={720} bind:value={deadlineHours} /></Field>
			<Field label="Warn this many hours before" hint="0 skips the warning."><NumberInput min={0} max={719} bind:value={warnHours} /></Field>
		</div>
		<Field label="Kick reason"><Input bind:value={gate.kickReason} maxlength={400} required /></Field>
		<Field label="Direct message on join" hint="Placeholders: {'{server} {duration}'}">
			<Textarea bind:value={gate.dmMessage} rows={6} />
		</Field>
		<Field label="Log channel"><ChannelSelect channels={data.bundle.channels} bind:value={gate.logChannelId} /></Field>
		<fieldset class="grid gap-2">
			<legend class="text-sm font-medium">Exempt roles</legend>
			{#each data.bundle.roles as role (role.id)}
				<Checkbox checked={gate.exemptRoleIds.includes(role.id)} onchange={() => toggleExempt(role.id)}>{role.name}</Checkbox>
			{/each}
		</fieldset>
		<div class="flex flex-wrap items-center gap-3">
			<Button type="submit" disabled={busy}>Save Application</Button>
			<Notice text={notice} />
		</div>
	</form>
</Card>

<Card size="xl" class="mb-6 flex-col gap-4 p-5">
	<Heading tag="h2" class="font-display text-xl font-semibold">Application message</Heading>
	<P class="text-sm">Save first. Members press Answer questions and fill out the form. Posting in the same channel updates the current message.</P>
	<Field label="Channel"><ChannelSelect channels={data.bundle.channels} bind:value={panelChannel} allowEmpty={false} /></Field>
	<Button type="button" onclick={postPanel}>Post Message</Button>
</Card>

<section class="grid gap-6">
	<div>
		<Heading tag="h2" class="mb-3 font-display text-xl font-semibold">Waiting</Heading>
		<div class="list grid gap-3">
			{#each data.pending as member (member.userId)}
				<Card size="xl" class="flex-col gap-1 p-4">
					<header class="flex items-baseline justify-between gap-3">
						<strong>{member.userName || member.userId}</strong>
						<span class="font-mono text-sm text-gray-500">{member.status}</span>
					</header>
					<P class="font-mono text-sm">Joined {when(member.joinedAt)} · deadline {when(member.deadlineAt)}</P>
				</Card>
			{:else}
				<P class="text-sm">Nobody is waiting.</P>
			{/each}
		</div>
	</div>
	<div>
		<Heading tag="h2" class="mb-3 font-display text-xl font-semibold">Removed this week</Heading>
		<div class="list grid gap-3">
			{#each data.recent as member (member.userId)}
				<Card size="xl" class="p-4">{member.userName || member.userId} · {when(member.deadlineAt)}</Card>
			{:else}
				<P class="text-sm">No recent kicks.</P>
			{/each}
		</div>
	</div>
</section>
