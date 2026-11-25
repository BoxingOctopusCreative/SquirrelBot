<script lang="ts">
	import { Button, Card, Fileupload, Input, P } from 'flowbite-svelte';
	import { untrack } from 'svelte';
	import { sendForm } from '#lib/api';
	import Field from '#lib/Field.svelte';
	import Notice from '#lib/Notice.svelte';
	import PageIntro from '#lib/PageIntro.svelte';
	import type { BotProfile, Bundle } from '#lib/types';

	let { data }: { data: { bundle: Bundle; profile: BotProfile } } = $props();
	let profile = $state<BotProfile>(untrack(() => ({ ...data.profile })));
	let nick = $state(untrack(() => data.profile.nick));
	let clearAvatar = $state(false);
	let preview = $state('');
	let notice = $state('');
	let busy = $state(false);

	function avatarInput(form: HTMLFormElement | null): HTMLInputElement | null {
		const input = form?.elements.namedItem('avatar');
		return input instanceof HTMLInputElement ? input : null;
	}

	let shown = $derived(
		preview || (clearAvatar ? profile.defaultAvatarUrl : profile.avatarUrl || profile.defaultAvatarUrl)
	);
	let username = $derived(profile.username || 'the bot');

	function pick(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const next = input.files?.[0];
		if (preview) URL.revokeObjectURL(preview);
		preview = '';
		if (!next) return;
		if (next.size > 1024 * 1024) {
			notice = 'Pictures must be 1 MB or smaller.';
			input.value = '';
			return;
		}
		notice = '';
		clearAvatar = false;
		preview = URL.createObjectURL(next);
	}

	function useUsualPicture(event: MouseEvent) {
		if (preview) URL.revokeObjectURL(preview);
		preview = '';
		clearAvatar = true;
		const input = avatarInput((event.currentTarget as HTMLButtonElement).form);
		if (input) input.value = '';
	}

	async function save(event: SubmitEvent) {
		event.preventDefault();
		const form = event.currentTarget as HTMLFormElement;
		busy = true;
		notice = '';
		try {
			profile = await sendForm<BotProfile>(`/api/guilds/${data.bundle.guild.id}/profile`, 'PUT', new FormData(form));
			if (preview) URL.revokeObjectURL(preview);
			preview = '';
			clearAvatar = false;
			nick = profile.nick;
			const input = avatarInput(form);
			if (input) input.value = '';
			notice = 'Saved. Profile updated.';
		} catch (err) {
			notice = err instanceof Error ? err.message : 'Could not save.';
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>Profile · SquirrelBot</title></svelte:head>
<PageIntro kicker="This server" title="Profile">
	<P>
		Members of this server see this name and picture. The username stays {username} on every server. The bot needs
		Change Nickname to set the name. Use Add a server if an older invite is missing that permission.
	</P>
</PageIntro>
<Card size="xl" class="flex-col p-5">
	<form class="grid gap-4" onsubmit={save}>
		<div class="flex items-center gap-4">
			{#if shown}
				<img class="h-16 w-16 rounded-full object-cover" src={shown} alt="Avatar for this server" />
			{:else}
				<div class="h-16 w-16 rounded-full bg-gray-200 dark:bg-gray-600" role="img" aria-label="No avatar yet"></div>
			{/if}
			<div>
				<P>{nick.trim() || username}</P>
				<P class="text-sm">Shown in the member list, chat, and voice.</P>
			</div>
		</div>
		<Field label="Display name" hint="Leave this blank to use the username.">
			<Input name="nick" bind:value={nick} maxlength={32} autocomplete="off" />
		</Field>
		<Field label="Avatar" hint="PNG, JPEG, or GIF. Up to 1 MB.">
			<Fileupload name="avatar" accept="image/png,image/jpeg,image/gif" onchange={pick} />
		</Field>
		<input type="hidden" name="clearAvatar" value={clearAvatar ? '1' : '0'} />
		<div class="flex flex-wrap items-center gap-3">
			<Button type="submit" disabled={busy}>Save Profile</Button>
			{#if (profile.customAvatar || preview) && !clearAvatar}
				<Button color="alternative" type="button" disabled={busy} onclick={useUsualPicture}>Use the Usual Picture</Button>
			{/if}
			{#if clearAvatar}<span class="text-sm text-gray-500">Save profile to use the usual picture.</span>{/if}
			<Notice text={notice} />
		</div>
	</form>
</Card>
