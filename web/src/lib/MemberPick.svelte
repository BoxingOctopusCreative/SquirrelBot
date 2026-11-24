<script lang="ts">
	import { Input, Label } from 'flowbite-svelte';
	import { onMount, tick } from 'svelte';
	import { sendJSON } from '#lib/api';
	import type { MemberHit } from '#lib/types';

	let {
		guildId,
		userId = $bindable(''),
		userName = $bindable('')
	}: {
		guildId: string;
		userId?: string;
		userName?: string;
	} = $props();

	let members = $state<MemberHit[]>([]);
	let query = $state('');
	let open = $state(false);
	let typing = $state(false);
	let ready = $state(false);
	let loadError = $state('');
	let active = $state(0);

	const listId = $derived(`member-list-${guildId}`);
	const inputId = $derived(`member-query-${guildId}`);
	const shown = $derived.by(() => {
		const needle = query.trim().toLowerCase();
		if (!typing || !needle) return members;
		return members.filter(
			(member) =>
				member.displayName.toLowerCase().includes(needle) ||
				member.username.toLowerCase().includes(needle)
		);
	});
	const current = $derived(shown.length === 0 ? -1 : Math.min(active, shown.length - 1));

	onMount(() => {
		let gone = false;
		sendJSON<{ members: MemberHit[] }>(`/api/guilds/${guildId}/members`, 'GET')
			.then((body) => {
				if (gone) return;
				members = body.members;
				ready = true;
			})
			.catch((err) => {
				if (gone) return;
				loadError = err instanceof Error ? err.message : 'Could not load members.';
				ready = true;
			});
		return () => {
			gone = true;
		};
	});

	function choose(member: MemberHit) {
		userId = member.id;
		userName = member.displayName;
		query = member.displayName;
		typing = false;
		open = false;
	}

	function onInput() {
		typing = true;
		open = true;
		active = 0;
		userId = '';
		userName = '';
	}

	function onFocus() {
		open = true;
		typing = false;
		const index = members.findIndex((member) => member.id === userId);
		active = index >= 0 ? index : 0;
	}

	async function onKeydown(event: KeyboardEvent) {
		if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
			event.preventDefault();
			open = true;
			if (shown.length === 0) return;
			if (current < 0) {
				active = event.key === 'ArrowDown' ? 0 : shown.length - 1;
			} else {
				const step = event.key === 'ArrowDown' ? 1 : -1;
				active = (current + step + shown.length) % shown.length;
			}
			await tick();
			const picked = shown[Math.min(active, shown.length - 1)];
			document.getElementById(`member-opt-${picked.id}`)?.scrollIntoView({ block: 'nearest' });
			return;
		}
		if (event.key === 'Escape') {
			open = false;
			return;
		}
		if (event.key === 'Enter' && open && current >= 0) {
			event.preventDefault();
			choose(shown[current]);
		}
	}

	function onWindowClick(event: MouseEvent) {
		const target = event.target;
		if (!(target instanceof Element) || !target.closest('.member-field')) open = false;
	}
</script>

<svelte:window onclick={onWindowClick} />

<div class="member-field grid gap-1.5">
	<Label id="{listId}-label" for={inputId}>Find a member</Label>
	<div class="relative">
		<Input
			id={inputId}
			role="combobox"
			aria-autocomplete="list"
			aria-expanded={open}
			aria-controls={listId}
			aria-labelledby="{listId}-label"
			aria-activedescendant={open && current >= 0 ? `member-opt-${shown[current].id}` : undefined}
			bind:value={query}
			placeholder="Name"
			autocomplete="off"
			onfocus={onFocus}
			oninput={onInput}
			onkeydown={onKeydown}
		/>
		{#if open}
			<ul
				class="absolute z-20 mt-1 max-h-64 w-full overflow-auto rounded-lg border border-gray-200 bg-white p-1 dark:border-gray-600 dark:bg-gray-800"
				id={listId}
				role="listbox"
			>
				{#if !ready}
					<li class="px-3 py-2 text-sm text-gray-500">Loading members.</li>
				{:else if loadError}
					<li class="px-3 py-2 text-sm text-gray-500" role="status">{loadError}</li>
				{:else if shown.length === 0}
					<li class="px-3 py-2 text-sm text-gray-500">No members match.</li>
				{:else}
					{#each shown as member, index (member.id)}
						<li>
							<button
								id="member-opt-{member.id}"
								type="button"
								role="option"
								aria-selected={index === current}
								class="flex w-full items-baseline justify-between gap-2 rounded px-3 py-2 text-left hover:bg-gray-100 dark:hover:bg-gray-700"
								onmousedown={(event) => event.preventDefault()}
								onmouseenter={() => (active = index)}
								onclick={() => choose(member)}
							>
								<span>{member.displayName}</span>
								{#if member.username.toLowerCase() !== member.displayName.toLowerCase()}
									<span class="font-mono text-sm text-gray-500">@{member.username}</span>
								{/if}
							</button>
						</li>
					{/each}
				{/if}
			</ul>
		{/if}
	</div>
	{#if loadError && !open}<small role="status">{loadError}</small>{/if}
</div>
