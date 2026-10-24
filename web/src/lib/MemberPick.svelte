<script lang="ts">
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

<div class="field member-field">
	<span id="{listId}-label">Find a member</span>
	<div class="member-pick">
		<input
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
			<ul class="member-menu" id={listId} role="listbox">
				{#if !ready}
					<li class="hint">Loading members.</li>
				{:else if loadError}
					<li class="hint">{loadError}</li>
				{:else if shown.length === 0}
					<li class="hint">No members match.</li>
				{:else}
					{#each shown as member, index (member.id)}
						<li>
							<button
								id="member-opt-{member.id}"
								type="button"
								role="option"
								aria-selected={index === current}
								onmousedown={(event) => event.preventDefault()}
								onmouseenter={() => (active = index)}
								onclick={() => choose(member)}
							>
								<span>{member.displayName}</span>
								{#if member.username.toLowerCase() !== member.displayName.toLowerCase()}
									<span class="meta">@{member.username}</span>
								{/if}
							</button>
						</li>
					{/each}
				{/if}
			</ul>
		{/if}
	</div>
	{#if loadError && !open}<small>{loadError}</small>{/if}
</div>
