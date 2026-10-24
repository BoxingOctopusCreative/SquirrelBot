<script lang="ts">
	import { refreshAll } from '$app/navigation';
	import { untrack } from 'svelte';
	import { sendJSON, when } from '#lib/api';
	import ChannelSelect from '#lib/ChannelSelect.svelte';
	import CreateChannel from '#lib/CreateChannel.svelte';
	import Field from '#lib/Field.svelte';
	import RoleSelect from '#lib/RoleSelect.svelte';
	import type { Announcement, AnnouncementTemplate, Bundle } from '#lib/types';

	let { data }: { data: { bundle: Bundle; announcements: Announcement[]; templates: AnnouncementTemplate[] } } =
		$props();
	let channelId = $state(untrack(() => data.bundle.channels[0]?.id ?? ''));
	let title = $state('');
	let body = $state('');
	let pingRoleId = $state('');
	let schedule = $state('');
	let notice = $state('');
	let busy = $state(false);

	let templateID = $state<number | null>(null);
	let templateName = $state('');
	let templateTitle = $state('');
	let templateBody = $state('');
	let templateValues = $state('');
	let templateNotice = $state('');
	let templateBusy = $state(false);

	function parseVars(text: string): Record<string, string> {
		const vars: Record<string, string> = {};
		for (const line of text.split('\n')) {
			const trimmed = line.trim();
			if (!trimmed) continue;
			const eq = trimmed.indexOf('=');
			if (eq <= 0) throw new Error('Write each value as name=text.');
			vars[trimmed.slice(0, eq).trim()] = trimmed.slice(eq + 1).trim();
		}
		return vars;
	}

	async function publish() {
		busy = true;
		notice = '';
		try {
			await sendJSON(`/api/guilds/${data.bundle.guild.id}/announcements`, 'POST', {
				channelId,
				title,
				body,
				pingRoleId,
				scheduledAt: schedule ? new Date(schedule).toISOString() : null
			});
			const scheduled = schedule;
			title = '';
			body = '';
			schedule = '';
			notice = scheduled ? 'Scheduled.' : 'Sent.';
			await refreshAll();
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not send.';
		} finally {
			busy = false;
		}
	}

	async function remove(id: number) {
		notice = '';
		try {
			await sendJSON(`/api/guilds/${data.bundle.guild.id}/announcements/${id}`, 'DELETE');
			await refreshAll();
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not delete.';
		}
	}

	async function sendNow(id: number) {
		notice = '';
		try {
			await sendJSON(`/api/guilds/${data.bundle.guild.id}/announcements/${id}/send`, 'POST');
			notice = 'Sent.';
			await refreshAll();
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not send.';
		}
	}

	function editTemplate(item: AnnouncementTemplate) {
		templateID = item.id;
		templateName = item.name;
		templateTitle = item.title;
		templateBody = item.body;
		templateNotice = '';
	}

	function clearTemplate() {
		templateID = null;
		templateName = '';
		templateTitle = '';
		templateBody = '';
	}

	async function saveTemplate() {
		templateBusy = true;
		templateNotice = '';
		const path = `/api/guilds/${data.bundle.guild.id}/announcement-templates`;
		try {
			if (templateID === null) {
				await sendJSON(path, 'POST', { name: templateName, title: templateTitle, body: templateBody });
			} else {
				await sendJSON(`${path}/${templateID}`, 'PUT', {
					name: templateName,
					title: templateTitle,
					body: templateBody
				});
			}
			clearTemplate();
			templateNotice = 'Template saved.';
			await refreshAll();
		} catch (err) {
			templateNotice = err instanceof Error ? err.message : 'Could not save the template.';
		} finally {
			templateBusy = false;
		}
	}

	async function useTemplate(id: number) {
		templateNotice = '';
		notice = '';
		try {
			const rendered = await sendJSON<{ title: string; body: string }>(
				`/api/guilds/${data.bundle.guild.id}/announcement-templates/${id}/render`,
				'POST',
				{ vars: parseVars(templateValues) }
			);
			title = rendered.title;
			body = rendered.body;
			notice = 'Filled from the template. Send it when it looks right.';
		} catch (err) {
			templateNotice = err instanceof Error ? err.message : 'Could not use that template.';
		}
	}

	async function deleteTemplate(id: number) {
		templateNotice = '';
		try {
			await sendJSON(`/api/guilds/${data.bundle.guild.id}/announcement-templates/${id}`, 'DELETE');
			if (templateID === id) clearTemplate();
			await refreshAll();
		} catch (err) {
			templateNotice = err instanceof Error ? err.message : 'Could not delete the template.';
		}
	}
</script>

<svelte:head><title>Announcements · SquirrelBot</title></svelte:head>
<p class="kicker">Mod team</p>
<h1>Announcements</h1>
<p class="sub">Posts go out as an embed from SquirrelBot, with your name in the footer.</p>

<form class="panel" onsubmit={(event) => { event.preventDefault(); publish(); }}>
	<Field label="Channel"><ChannelSelect channels={data.bundle.channels} bind:value={channelId} allowEmpty={false} /></Field>
	<Field label="Title"><input bind:value={title} maxlength="256" required /></Field>
	<Field label="Body"><textarea bind:value={body} maxlength="4000" required></textarea></Field>
	<div class="split">
		<Field label="Ping a role" hint="Leave empty to post quietly.">
			<RoleSelect roles={data.bundle.roles} bind:value={pingRoleId} />
		</Field>
		<Field label="Send later" hint="Leave empty to send now.">
			<input type="datetime-local" bind:value={schedule} />
		</Field>
	</div>
	<div class="actions">
		<button type="submit" disabled={busy}>{schedule ? 'Schedule' : 'Send now'}</button>
		<CreateChannel
			guildId={data.bundle.guild.id}
			purpose="announcements"
			oncreated={(channel) => {
				channelId = channel.id;
			}}
		/>
		{#if notice}<span class="banner">{notice}</span>{/if}
	</div>
</form>

<section class="panel">
	<p class="kicker">Templates</p>
	<p class="sub">
		Save a canned announcement. The title and body can be plain text, or <a href="https://shopify.github.io/liquid/" target="_blank" rel="noopener noreferrer">Liquid</a>.
	</p>
	<div class="liquid-help">
		<h2>Example</h2>
		<p>Title</p>
		<pre>{'{{ server | upcase }} downtime'}</pre>
		<p>Body</p>
		<pre>{'{{ server }} pauses at {{ when }}. Posted by {{ author }}.'}</pre>
		<p>Put <code>when=Tonight</code> in Values for the next use, then press Use. The title is the server name in capitals, then the word downtime. The body names the server, Tonight, and you.</p>
		<h2>Built-in variables</h2>
		<dl>
			<div>
				<dt><code>{'{{ server }}'}</code></dt>
				<dd>The Discord server name. Write it in the title or the body. It is filled in when you press Use.</dd>
			</div>
			<div>
				<dt><code>{'{{ author }}'}</code></dt>
				<dd>Your Discord username. Write it in the title or the body. It is filled in when you press Use.</dd>
			</div>
		</dl>
		<p>Any other name, such as <code>when</code>, comes from the values box. Write one <code>name=text</code> line for each.</p>
	</div>
	<hr class="divider" />
	<form
		onsubmit={(event) => {
			event.preventDefault();
			saveTemplate();
		}}
	>
		<Field label="Name"><input bind:value={templateName} maxlength="40" required /></Field>
		<Field label="Title"><input bind:value={templateTitle} maxlength="256" required /></Field>
		<Field label="Body"><textarea bind:value={templateBody} maxlength="4000" required></textarea></Field>
		<div class="actions">
			<button type="submit" disabled={templateBusy}>{templateID === null ? 'Save template' : 'Update template'}</button>
			{#if templateID !== null}<button class="quiet" type="button" onclick={clearTemplate}>Cancel</button>{/if}
			{#if templateNotice}<span class="banner">{templateNotice}</span>{/if}
		</div>
	</form>
	<Field label="Values for the next use" hint="One extra value per line, such as when=Tonight. Leave this empty when the template only needs server and author.">
		<textarea class="values" bind:value={templateValues}></textarea>
	</Field>
	<section class="list">
		{#each data.templates as item (item.id)}
			<article class="item">
				<header>
					<strong>{item.name}</strong>
					<span class="meta">{when(item.updatedAt)}</span>
				</header>
				<p>{item.title}</p>
				<p class="meta">{item.body}</p>
				<div class="actions">
					<button type="button" onclick={() => useTemplate(item.id)}>Use</button>
					<button class="quiet" type="button" onclick={() => editTemplate(item)}>Edit</button>
					<button class="quiet" type="button" onclick={() => deleteTemplate(item.id)}>Delete</button>
				</div>
			</article>
		{:else}
			<p class="hint">No templates yet.</p>
		{/each}
	</section>
</section>

<section class="list">
	{#each data.announcements as item (item.id)}
		<article class="item">
			<header>
				<strong>{item.title}</strong>
				<span class="meta">{item.status}</span>
			</header>
			<p>{item.body}</p>
			<p class="meta">{item.authorName} · {when(item.sentAt || item.scheduledAt || item.createdAt)}{item.error ? ` · ${item.error}` : ''}</p>
			{#if item.status !== 'sent'}
				<div class="actions">
					<button type="button" onclick={() => sendNow(item.id)}>Send now</button>
					<button class="quiet" type="button" onclick={() => remove(item.id)}>Delete</button>
				</div>
			{/if}
		</article>
	{:else}
		<p class="hint">No announcements yet.</p>
	{/each}
</section>
