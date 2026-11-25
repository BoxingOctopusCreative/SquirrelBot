<script lang="ts">
	import { Card, Heading, P } from 'flowbite-svelte';
	import { when } from '#lib/api';
	import PageIntro from '#lib/PageIntro.svelte';
	import type { ModCase } from '#lib/types';

	let { data }: { data: { cases: ModCase[] } } = $props();
</script>

<svelte:head><title>Moderation Log · SquirrelBot</title></svelte:head>
<PageIntro kicker="Record" title="Moderation Log">
	<P>Warns, kicks, bans, timeouts, admin firecall access, and application kicks.</P>
</PageIntro>
<section class="list grid gap-3">
	{#each data.cases as item (item.id)}
		<Card size="xl" class="flex-col gap-1 p-4">
			<header class="flex items-baseline justify-between gap-3">
				<Heading tag="h2" class="font-display text-lg font-semibold">#{item.caseNumber} {item.action === 'gate_kick' ? 'application kick' : item.action}</Heading>
				<span class="font-mono text-sm text-gray-500">{when(item.createdAt)}</span>
			</header>
			<P>{item.targetName || item.targetId}</P>
			<P class="font-mono text-sm">{item.moderatorName} · {item.reason || 'No reason given.'}</P>
		</Card>
	{:else}
		<P class="text-sm">No moderation actions yet.</P>
	{/each}
</section>
