<script lang="ts">
	import { when } from '#lib/api';
	import type { ModCase } from '#lib/types';

	let { data }: { data: { cases: ModCase[] } } = $props();
</script>

<svelte:head><title>Cases · SquirrelBot</title></svelte:head>
<p class="kicker">Record</p>
<h1>Cases</h1>
<p class="sub">Warns, kicks, bans, timeouts, admin firecall access, and application kicks.</p>
<section class="list">
	{#each data.cases as item (item.id)}
		<article class="item">
			<header>
				<strong>#{item.caseNumber} {item.action === 'gate_kick' ? 'application kick' : item.action}</strong>
				<span class="meta">{when(item.createdAt)}</span>
			</header>
			<p>{item.targetName || item.targetId}</p>
			<p class="meta">{item.moderatorName} · {item.reason || 'No reason given.'}</p>
		</article>
	{:else}
		<p class="hint">No cases yet.</p>
	{/each}
</section>
