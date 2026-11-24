<script lang="ts">
	import { Avatar, Button, Card, Heading, P } from 'flowbite-svelte';
	import type { GuildSummary } from '#lib/types';

	let { data }: { data: { guilds: GuildSummary[] } } = $props();
</script>

<svelte:head><title>Servers · SquirrelBot</title></svelte:head>

{#if data.guilds.length === 0}
	<main id="main" class="mx-auto max-w-xl p-8" tabindex="-1">
		<Heading tag="h1" class="mb-3 font-display text-3xl font-semibold">No servers yet</Heading>
		<P>You will see servers where SquirrelBot is a member and you have Manage Server or a dashboard role.</P>
		<Button href="/auth/discord/invite" class="mt-4">Add SquirrelBot to a server</Button>
	</main>
{:else}
	<main id="main" class="mx-auto max-w-5xl p-6" tabindex="-1">
		<Heading tag="h1" class="mb-4 font-display text-3xl font-semibold">Servers</Heading>
		<section class="grid grid-cols-[repeat(auto-fill,minmax(16rem,1fr))] gap-3">
			{#each data.guilds as guild (guild.id)}
				<Card href={`/app/${guild.id}`} size="xl" class="flex-row items-center gap-3 p-4">
					{#if guild.iconUrl}
						<Avatar src={guild.iconUrl} alt="" size="sm" />
					{:else}
						<span class="inline-block h-8 w-8 shrink-0 rounded-full bg-gray-200 dark:bg-gray-600" aria-hidden="true"></span>
					{/if}
					<strong class="underline decoration-from-font underline-offset-4">{guild.name}</strong>
				</Card>
			{/each}
		</section>
	</main>
{/if}
