<script lang="ts">
	import { onMount } from 'svelte';

	const storageKey = 'squirrelbot-theme';

	let dark = $state(false);

	function storedTheme(): 'dark' | 'light' | null {
		const stored = localStorage.getItem(storageKey);
		return stored === 'dark' || stored === 'light' ? stored : null;
	}

	function systemDark(): boolean {
		return window.matchMedia('(prefers-color-scheme: dark)').matches;
	}

	function paint(mode: 'dark' | 'light' | null) {
		if (mode === null) delete document.documentElement.dataset.theme;
		else document.documentElement.dataset.theme = mode;
		dark = mode === null ? systemDark() : mode === 'dark';
	}

	function toggle() {
		const next = dark ? 'light' : 'dark';
		localStorage.setItem(storageKey, next);
		paint(next);
	}

	onMount(() => {
		paint(storedTheme());
		const media = window.matchMedia('(prefers-color-scheme: dark)');
		const onChange = () => {
			if (storedTheme() === null) paint(null);
		};
		media.addEventListener('change', onChange);
		return () => media.removeEventListener('change', onChange);
	});
</script>

<button class="quiet theme-toggle" type="button" aria-pressed={dark ? 'true' : 'false'} onclick={toggle}>
	Dark mode
</button>
