<script lang="ts">
	import { Button, Card, Input, P, Select, Textarea } from 'flowbite-svelte';
	import { refreshAll } from '$app/navigation';
	import { sendJSON } from '#lib/api';
	import Field from '#lib/Field.svelte';
	import Notice from '#lib/Notice.svelte';
	import PageIntro from '#lib/PageIntro.svelte';
	import type { Bundle, CustomCommand } from '#lib/types';

	let { data }: { data: { bundle: Bundle; commands: CustomCommand[] } } = $props();
	let name = $state('');
	let response = $state('');
	let roleIds = $state<string[]>([]);
	let editing = $state(false);
	let notice = $state('');

	function scriptSource(command: CustomCommand): string {
		if (command.kind === 'lua') return command.response;
		const body = command.response ?? '';
		let eq = '';
		while (body.includes(`]${eq}]`)) eq += '=';
		return `return [${eq}[${body}]${eq}]`;
	}

	function edit(command: CustomCommand) {
		editing = true;
		name = command.name;
		response = scriptSource(command);
		roleIds = [...(command.roleIds ?? [])];
		notice = '';
	}

	function clearForm() {
		editing = false;
		name = '';
		response = '';
		roleIds = [];
	}

	async function save(event: SubmitEvent) {
		event.preventDefault();
		notice = '';
		try {
			await sendJSON(`/api/guilds/${data.bundle.guild.id}/commands`, 'POST', { name, response, roleIds });
			clearForm();
			notice = 'Saved. Command updated.';
			await refreshAll();
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not save.';
		}
	}

	async function remove(command: string) {
		notice = '';
		try {
			await sendJSON(`/api/guilds/${data.bundle.guild.id}/commands/${command}`, 'DELETE');
			if (name === command) clearForm();
			await refreshAll();
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not delete.';
		}
	}
</script>

<svelte:head><title>Commands · SquirrelBot</title></svelte:head>
<PageIntro kicker="Shortcuts" title="Commands">
	<P>
		Members type {data.bundle.guild.prefix}name, then any arguments. A script can branch, build a reply, and change a role you select here. Mentions in a reply are not pinged.
	</P>
</PageIntro>
<Card size="xl" class="mb-6 flex-col p-5">
	<form class="grid gap-4" onsubmit={save}>
		<Field label="Name"><Input bind:value={name} required maxlength={32} pattern={'[a-z0-9_-]+'} /></Field>
		<Field label="Script" hint="Lua 5.1. Return a string, or call reply.">
			<Textarea class="min-h-64 font-mono text-sm" bind:value={response} maxlength={8000} required spellcheck={false} rows={14} />
		</Field>
		<Field label="Roles this command may change" hint="addRole and removeRole only accept these. Use the role name or its id.">
			<Select multiple placeholder="" bind:value={roleIds}>
				{#each data.bundle.roles as role (role.id)}
					<option value={role.id} disabled={role.managed}>{role.name}</option>
				{/each}
			</Select>
		</Field>
		<div class="grid gap-2 text-sm">
			<h2 class="font-display text-lg font-semibold">What a script can do</h2>
			<P><code class="font-mono">reply(text)</code> sends one message. Returning a string does the same when you do not call reply.</P>
			<P>
				<code class="font-mono">liquid(template)</code> fills <code class="font-mono">{'{{ author.name }}'}</code>, <code class="font-mono">{'{{ author.id }}'}</code>,
				<code class="font-mono">{'{{ author.display_name }}'}</code>, <code class="font-mono">{'{{ args }}'}</code>, <code class="font-mono">{'{{ guild.name }}'}</code>, and
				<code class="font-mono">{'{{ prefix }}'}</code>.
			</P>
			<P><code class="font-mono">react(emoji)</code> and <code class="font-mono">delete()</code> act on the message that ran the command.</P>
			<P><code class="font-mono">addRole(name)</code> and <code class="font-mono">removeRole(name)</code> change a role selected above. A name or an id both work.</P>
			<h2 class="mt-2 font-display text-lg font-semibold">Example</h2>
			<pre class="overflow-x-auto rounded-lg bg-gray-100 p-3 font-mono text-sm whitespace-pre-wrap dark:bg-gray-900">{`if args[1] == "join" then
  addRole("Member")
  reply(liquid("Welcome, {{ author.name }}"))
  react("✅")
else
  reply("Try " .. prefix .. "join")
end`}</pre>
		</div>
		<div class="flex flex-wrap items-center gap-3">
			<Button type="submit">{editing ? 'Update command' : 'Save command'}</Button>
			{#if editing}<Button color="alternative" type="button" onclick={clearForm}>Cancel</Button>{/if}
			<Notice text={notice} />
		</div>
	</form>
</Card>
<section class="list grid gap-3">
	{#each data.commands as command (command.name)}
		<Card size="xl" class="flex-col gap-2 p-4">
			<header class="flex items-baseline justify-between gap-3">
				<strong>{data.bundle.guild.prefix}{command.name}</strong>
				<span class="font-mono text-sm text-gray-500">{command.kind === 'lua' ? 'Script' : 'Text'}</span>
			</header>
			<pre class="overflow-x-auto font-mono text-sm whitespace-pre-wrap">{scriptSource(command)}</pre>
			<div class="flex flex-wrap gap-2">
				<Button color="alternative" size="sm" type="button" onclick={() => edit(command)}>Edit</Button>
				<Button color="red" size="sm" type="button" onclick={() => remove(command.name)}>Delete</Button>
			</div>
		</Card>
	{:else}
		<P class="text-sm">No custom commands yet.</P>
	{/each}
</section>
