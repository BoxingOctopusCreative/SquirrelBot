<script lang="ts">
	import { page } from '$app/state';
	import Mark from '#lib/Mark.svelte';

	let problem = $derived(page.url.searchParams.get('error'));
</script>

<svelte:head><title>SquirrelBot</title></svelte:head>

<main class="login">
	<section class="login-copy">
		<div>
			<Mark size={42} />
			<h1 class="word">SquirrelBot</h1>
			<p class="lede">Moderation for the people who keep a Discord server standing.</p>
			<ul class="jobs">
				<li><strong>Mod team announcements</strong><span>Post once, in the channel you choose, now or later.</span></li>
				<li><strong>Temporary staff</strong><span>Lend a mod or admin role, then take it back when the time ends.</span></li>
				<li><strong>Server application</strong><span>Kick members who do not finish the questions in time.</span></li>
			</ul>
		</div>
		{#if problem}<p class="banner">Discord did not finish sign-in. Try again.</p>{/if}
	</section>
	<aside class="login-aside">
		<div class="login-actions">
			<a class="signin" href="/auth/discord">Continue with Discord</a>
			<p class="hint">Discord asks you to sign in, then add SquirrelBot to a server you manage.</p>
		</div>
	</aside>
</main>
