<script lang="ts">
	import type { GuildSummary } from '#lib/types';

	let { data }: { data: { guilds: GuildSummary[] } } = $props();
</script>

<svelte:head><title>Servers · SquirrelBot</title></svelte:head>

{#if data.guilds.length === 0}
	<main id="main" class="empty" tabindex="-1">
		<h1>No servers yet</h1>
		<p>You will see servers where SquirrelBot is a member and you have Manage Server or a dashboard role.</p>
		<p><a class="button" href="/auth/discord/invite">Add SquirrelBot to a server</a></p>
	</main>
{:else}
	<main id="main" class="main" tabindex="-1">
		<h1>Servers</h1>
		<section class="guild-grid">
			{#each data.guilds as guild (guild.id)}
				<a class="guild-card" href={`/app/${guild.id}`}>
					{#if guild.iconUrl}
						<img class="guild-icon" src={guild.iconUrl} alt="" />
					{:else}
						<span class="guild-icon" aria-hidden="true"></span>
					{/if}
					<strong>{guild.name}</strong>
				</a>
			{/each}
		</section>
	</main>
{/if}
