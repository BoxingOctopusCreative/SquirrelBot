<script lang="ts">
	import type { GuildSummary } from '#lib/types';

	let { data }: { data: { guilds: GuildSummary[] } } = $props();
</script>

<svelte:head><title>Servers · SquirrelBot</title></svelte:head>

{#if data.guilds.length === 0}
	<section class="empty">
		<h1>No servers yet</h1>
		<p>You will see servers where you have Manage Server and SquirrelBot is a member.</p>
		<p><a class="button" href="/auth/discord">Add SquirrelBot to a server</a></p>
	</section>
{:else}
	<section class="guild-grid">
		{#each data.guilds as guild (guild.id)}
			<a class="guild-card" href={`/app/${guild.id}`}>
				{#if guild.iconUrl}
					<img class="guild-icon" src={guild.iconUrl} alt="" />
				{:else}
					<span class="guild-icon"></span>
				{/if}
				<strong>{guild.name}</strong>
			</a>
		{/each}
	</section>
{/if}
