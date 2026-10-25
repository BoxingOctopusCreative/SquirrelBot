<script lang="ts">
	import { refreshAll } from '$app/navigation';
	import { sendJSON } from '#lib/api';
	import Field from '#lib/Field.svelte';
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
			notice = 'Saved.';
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
<p class="kicker">Shortcuts</p>
<h1>Commands</h1>
<p class="sub">
	Members type {data.bundle.guild.prefix}name, then any arguments. A script can branch, build a reply, and change a role you select here. Mentions in a reply are not pinged.
</p>
<form class="panel" onsubmit={save}>
	<Field label="Name"><input bind:value={name} required maxlength="32" pattern="[a-z0-9_-]+" /></Field>
	<Field label="Script" hint="Lua 5.1. Return a string, or call reply.">
		<textarea class="script" bind:value={response} maxlength="8000" required spellcheck="false"></textarea>
	</Field>
	<Field label="Roles this command may change" hint="addRole and removeRole only accept these. Use the role name or its id.">
		<select multiple bind:value={roleIds}>
			{#each data.bundle.roles as role (role.id)}
				<option value={role.id} disabled={role.managed}>{role.name}</option>
			{/each}
		</select>
	</Field>
	<div class="liquid-help">
		<h2>What a script can do</h2>
		<p><code>reply(text)</code> sends one message. Returning a string does the same when you do not call reply.</p>
		<p>
			<code>liquid(template)</code> fills <code>{'{{ author.name }}'}</code>, <code>{'{{ author.id }}'}</code>,
			<code>{'{{ author.display_name }}'}</code>, <code>{'{{ args }}'}</code>, <code>{'{{ guild.name }}'}</code>, and
			<code>{'{{ prefix }}'}</code>.
		</p>
		<p><code>react(emoji)</code> and <code>delete()</code> act on the message that ran the command.</p>
		<p><code>addRole(name)</code> and <code>removeRole(name)</code> change a role selected above. A name or an id both work.</p>
		<h2>Example</h2>
		<pre>{`if args[1] == "join" then
  addRole("Member")
  reply(liquid("Welcome, {{ author.name }}"))
  react("✅")
else
  reply("Try " .. prefix .. "join")
end`}</pre>
	</div>
	<div class="actions">
		<button type="submit">{editing ? 'Update command' : 'Save command'}</button>
		{#if editing}<button class="quiet" type="button" onclick={clearForm}>Cancel</button>{/if}
		{#if notice}<span class="banner">{notice}</span>{/if}
	</div>
</form>
<section class="list">
	{#each data.commands as command (command.name)}
		<article class="item">
			<header>
				<strong>{data.bundle.guild.prefix}{command.name}</strong>
				<span class="meta">{command.kind === 'lua' ? 'Script' : 'Text'}</span>
			</header>
			<pre>{scriptSource(command)}</pre>
			<div class="actions">
				<button class="quiet" type="button" onclick={() => edit(command)}>Edit</button>
				<button class="quiet" type="button" onclick={() => remove(command.name)}>Delete</button>
			</div>
		</article>
	{:else}
		<p class="hint">No custom commands yet.</p>
	{/each}
</section>
