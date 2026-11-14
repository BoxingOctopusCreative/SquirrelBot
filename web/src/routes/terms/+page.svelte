<script lang="ts">
	import Legal from '#lib/Legal.svelte';
</script>

<Legal title="Terms of Service">
	<p>SquirrelBot is a Discord moderation bot and a dashboard for the people who run a server. Boxing Octopus publishes SquirrelBot. Signing in, adding the bot to a server, or using a command or button the bot provides means you agree to these terms and to the <a href="/privacy">Privacy Policy</a>.</p>

	<h2>What the bot does</h2>
	<p>Staff can warn, kick, ban, time out, and keep a case history. The bot can post announcements, lend a role for a limited time, ask new members to finish a server application, write logs into a channel the server chooses, run automod, offer reaction roles, repost starred messages, take anonymous confessions, open support tickets, and answer custom commands. Moderators on that server choose which of these are turned on.</p>

	<h2>Discord accounts</h2>
	<p>You sign in with Discord. Opening a server in the dashboard requires a Discord account and permission on that server. Installing the bot requires Manage Server. <a href="https://discord.com/terms">Discord's terms</a>, <a href="https://discord.com/guidelines">community guidelines</a>, and <a href="https://discord.com/developers/docs/policies-and-agreements/developer-terms-of-service">developer terms</a> also apply to your account, the server, and the bot.</p>

	<h2>Server applications</h2>
	<p>A server can ask new members to answer a short form. The first question is "What is your age?". An answer under 18 removes that member from the server. A case records the removal with the reason "Under the age requirement". The age number is not stored. The other answers are checked so the bot knows the form was finished, and they are not saved as a profile.</p>

	<h2>Confessions</h2>
	<p>A confession goes to moderators before it is posted. The public post leaves the author's name off. Moderators can see who sent it, in the review channel and in the dashboard's records. You can attach one file. That file is posted in Discord with the confession. Send only text and files you have the right to share.</p>

	<h2>Support tickets</h2>
	<p>A ticket is a private channel between you and the support role that server chose. The subject, your request, and the messages in that channel are saved when the ticket is closed. Staff of that server can read the transcript in the dashboard. Send only text and files you have the right to share.</p>

	<h2>Acceptable use</h2>
	<p>Use the bot for the server that installed it. Do not use it to harass someone, to break into an account or a server, or to get around a limit the bot enforces. Follow the rules of the server you are in. Follow Discord's rules when you use the bot.</p>

	<h2>Stopping</h2>
	<p>Sign out of the dashboard when you are done. A server can remove the bot at any time. Removing the bot stops new actions. Records already stored for that server stay until the operator deletes them. Write to us if you want those records removed.</p>

	<h2>The software</h2>
	<p>The SquirrelBot source code is available under the Mozilla Public License 2.0. The dashboard and bot are provided as they run today. Features can change. We can refuse a use that breaks these terms or Discord's rules.</p>

	<h2>Responsibility</h2>
	<p>Moderation actions are taken by the server's staff, or by rules that staff turned on. Boxing Octopus does not moderate your server. To the extent the law allows, Boxing Octopus is not liable for a lost message, a moderation action, or downtime. The service is provided without a warranty of any kind.</p>

	<h2>Changes</h2>
	<p>When these terms change, the date at the top of this page changes. Using SquirrelBot after that date means you accept the updated terms.</p>

	<h2>Contact</h2>
	<p>Questions about these terms can go to <a href="https://github.com/BoxingOctopus/squirrelbot">github.com/BoxingOctopus/squirrelbot</a>.</p>
</Legal>
