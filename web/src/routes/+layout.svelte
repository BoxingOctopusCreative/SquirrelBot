<script lang="ts">
	import { afterNavigate } from '$app/navigation';
	import '../app.css';
	import type { Snippet } from 'svelte';

	let { children }: { children: Snippet } = $props();

	afterNavigate((navigation) => {
		if (navigation.type === 'enter') return;
		const heading = document.querySelector('main h1');
		if (!(heading instanceof HTMLElement)) return;
		heading.tabIndex = -1;
		heading.focus();
	});
</script>

<a class="skip" href="#main">Skip to main content</a>
{@render children()}
