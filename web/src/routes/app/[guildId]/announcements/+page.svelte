<script lang="ts">
	import { A, Button, Card, Heading, Hr, Input, P, Textarea } from 'flowbite-svelte';
	import { refreshAll } from '$app/navigation';
	import { untrack } from 'svelte';
	import { sendJSON, when } from '#lib/api';
	import ChannelSelect from '#lib/ChannelSelect.svelte';
	import CreateChannel from '#lib/CreateChannel.svelte';
	import Field from '#lib/Field.svelte';
	import Notice from '#lib/Notice.svelte';
	import PageIntro from '#lib/PageIntro.svelte';
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
<PageIntro kicker="Mod team" title="Announcements">
	<P>Posts go out as an embed from SquirrelBot, with your name in the footer.</P>
</PageIntro>

<Card size="xl" class="mb-6 flex-col p-5">
	<form class="grid gap-4" onsubmit={(event) => { event.preventDefault(); publish(); }}>
		<Field label="Channel"><ChannelSelect channels={data.bundle.channels} bind:value={channelId} allowEmpty={false} /></Field>
		<Field label="Title"><Input bind:value={title} maxlength={256} required /></Field>
		<Field label="Body"><Textarea bind:value={body} maxlength={4000} required rows={6} /></Field>
		<div class="grid gap-4 md:grid-cols-2">
			<Field label="Ping a role" hint="Leave empty to post quietly.">
				<RoleSelect roles={data.bundle.roles} bind:value={pingRoleId} />
			</Field>
			<Field label="Send later" hint="Leave empty to send now.">
				<Input type="datetime-local" bind:value={schedule} />
			</Field>
		</div>
		<div class="flex flex-wrap items-center gap-3">
			<Button type="submit" disabled={busy}>{schedule ? 'Schedule' : 'Send Now'}</Button>
			<CreateChannel
				guildId={data.bundle.guild.id}
				purpose="announcements"
				oncreated={(channel) => {
					channelId = channel.id;
				}}
			/>
			<Notice text={notice} />
		</div>
	</form>
</Card>

<Card size="xl" class="mb-6 flex-col gap-4 p-5">
	<p class="text-xs font-semibold tracking-widest text-primary-600 uppercase">Templates</p>
	<P>
		Save a canned announcement. The title and body can be plain text, or <A href="https://shopify.github.io/liquid/" target="_blank" rel="noopener noreferrer">Liquid</A>.
	</P>
	<div class="grid gap-2">
		<Heading tag="h2" class="font-display text-lg font-semibold">Example</Heading>
		<P>Title</P>
		<pre class="overflow-x-auto rounded-lg bg-gray-100 p-3 font-mono text-sm whitespace-pre-wrap dark:bg-gray-900">{'{{ server | upcase }} downtime'}</pre>
		<P>Body</P>
		<pre class="overflow-x-auto rounded-lg bg-gray-100 p-3 font-mono text-sm whitespace-pre-wrap dark:bg-gray-900">{'{{ server }} pauses at {{ when }}. Posted by {{ author }}.'}</pre>
		<P>Put <code class="font-mono">when=Tonight</code> in Values for the next use, then press Use. The title is the server name in capitals, then the word downtime. The body names the server, Tonight, and you.</P>
		<Heading tag="h2" class="mt-2 font-display text-lg font-semibold">Built-in variables</Heading>
		<dl class="grid gap-3">
			<div>
				<dt><code class="font-mono">{'{{ server }}'}</code></dt>
				<dd class="text-gray-600 dark:text-gray-300">The Discord server name. Write it in the title or the body. It is filled in when you press Use.</dd>
			</div>
			<div>
				<dt><code class="font-mono">{'{{ author }}'}</code></dt>
				<dd class="text-gray-600 dark:text-gray-300">Your Discord username. Write it in the title or the body. It is filled in when you press Use.</dd>
			</div>
		</dl>
		<P>Any other name, such as <code class="font-mono">when</code>, comes from the values box. Write one <code class="font-mono">name=text</code> line for each.</P>
	</div>
	<Hr />
	<form
		class="grid gap-4"
		onsubmit={(event) => {
			event.preventDefault();
			saveTemplate();
		}}
	>
		<Field label="Name"><Input bind:value={templateName} maxlength={40} required /></Field>
		<Field label="Title"><Input bind:value={templateTitle} maxlength={256} required /></Field>
		<Field label="Body"><Textarea bind:value={templateBody} maxlength={4000} required rows={6} /></Field>
		<div class="flex flex-wrap items-center gap-3">
			<Button type="submit" disabled={templateBusy}>{templateID === null ? 'Save Template' : 'Update Template'}</Button>
			{#if templateID !== null}<Button color="alternative" type="button" onclick={clearTemplate}>Cancel</Button>{/if}
			<Notice text={templateNotice} />
		</div>
	</form>
	<Field label="Values for the next use" hint="One extra value per line, such as when=Tonight. Leave this empty when the template only needs server and author.">
		<Textarea class="min-h-16" bind:value={templateValues} rows={3} />
	</Field>
	<section class="list grid gap-3">
		{#each data.templates as item (item.id)}
			<Card size="xl" class="flex-col gap-2 p-4">
				<header class="flex items-baseline justify-between gap-3">
					<strong>{item.name}</strong>
					<span class="font-mono text-sm text-gray-500">{when(item.updatedAt)}</span>
				</header>
				<P>{item.title}</P>
				<P class="font-mono text-sm">{item.body}</P>
				<div class="flex flex-wrap gap-2">
					<Button size="sm" type="button" onclick={() => useTemplate(item.id)}>Use</Button>
					<Button color="alternative" size="sm" type="button" onclick={() => editTemplate(item)}>Edit</Button>
					<Button color="red" size="sm" type="button" onclick={() => deleteTemplate(item.id)}>Delete</Button>
				</div>
			</Card>
		{:else}
			<P class="text-sm">No templates yet.</P>
		{/each}
	</section>
</Card>

<section class="list grid gap-3">
	{#each data.announcements as item (item.id)}
		<Card size="xl" class="flex-col gap-2 p-4">
			<header class="flex items-baseline justify-between gap-3">
				<strong>{item.title}</strong>
				<span class="font-mono text-sm text-gray-500">{item.status}</span>
			</header>
			<P>{item.body}</P>
			<P class="font-mono text-sm">{item.authorName} · {when(item.sentAt || item.scheduledAt || item.createdAt)}{item.error ? ` · ${item.error}` : ''}</P>
			{#if item.status !== 'sent'}
				<div class="flex flex-wrap gap-2">
					<Button size="sm" type="button" onclick={() => sendNow(item.id)}>Send Now</Button>
					<Button color="red" size="sm" type="button" onclick={() => remove(item.id)}>Delete</Button>
				</div>
			{/if}
		</Card>
	{:else}
		<P class="text-sm">No announcements yet.</P>
	{/each}
</section>
