<script lang="ts">
	import Moon from '@lucide/svelte/icons/moon';
	import Sun from '@lucide/svelte/icons/sun';
	import { Button } from 'flowbite-svelte';
	import { onMount } from 'svelte';

	const storageKey = 'squirrelbot-theme';

	let dark = $state(false);

	function storedTheme(): 'dark' | 'light' | null {
		const stored = localStorage.getItem(storageKey);
		return stored === 'dark' || stored === 'light' ? stored : null;
	}

	function prefersLight(): boolean {
		return window.matchMedia('(prefers-color-scheme: light)').matches;
	}

	function paint(mode: 'dark' | 'light' | null) {
		if (mode === null) delete document.documentElement.dataset.theme;
		else document.documentElement.dataset.theme = mode;
		dark = mode === 'dark' || (mode === null && !prefersLight());
		document.documentElement.classList.toggle('dark', dark);
	}

	function toggle() {
		const next = dark ? 'light' : 'dark';
		localStorage.setItem(storageKey, next);
		paint(next);
	}

	onMount(() => {
		paint(storedTheme());
		const media = window.matchMedia('(prefers-color-scheme: light)');
		const onChange = () => {
			if (storedTheme() === null) paint(null);
		};
		media.addEventListener('change', onChange);
		return () => media.removeEventListener('change', onChange);
	});
</script>

<Button
	color="alternative"
	aria-pressed={dark}
	aria-label={dark ? 'Use Light Mode' : 'Use Dark Mode'}
	class={dark
		? 'border-[#d08a4a]! bg-[#141821]! text-[#f6eadc]! hover:bg-[#1b2130]!'
		: 'border-[#d08a4a]! bg-[#f6eadc]! text-[#7a4518]! hover:bg-[#ecd3b6]!'}
	onclick={toggle}
>
	{#if dark}
		<Moon size={20} aria-hidden="true" />
	{:else}
		<Sun size={20} aria-hidden="true" />
	{/if}
</Button>
