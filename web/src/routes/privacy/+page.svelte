<script lang="ts">
	import Legal from '#lib/Legal.svelte';
</script>

<Legal title="Privacy Policy">
	<p>This policy describes the information SquirrelBot handles when you sign in to the dashboard or when the bot is used on a Discord server. Boxing Octopus publishes SquirrelBot. <a href="https://discord.com/privacy">Discord's privacy policy</a> covers your Discord account itself.</p>

	<h2>Signing in</h2>
	<p>Sign-in uses Discord OAuth with the <code>identify</code> and <code>guilds</code> scopes. Adding the bot also asks Discord for permission to install it. We receive your Discord user id, username, and avatar, plus the servers Discord includes in that response. We do not receive your Discord password.</p>
	<p>The dashboard stores an access token and a refresh token so it can stay signed in and list your servers. Both tokens are encrypted before they are saved. A session lasts 14 days. Signing out deletes that session.</p>

	<h2>Cookies</h2>
	<p>The site sets two cookies, and no others. Both are HttpOnly and SameSite Lax. They are marked Secure when the site is served over HTTPS.</p>
	<ul>
		<li><code>sb_oauth_state</code> lasts about 10 minutes. It checks that the Discord sign-in response matches the request that started it.</li>
		<li><code>sb_session</code> lasts up to 14 days. It is the signed-in session.</li>
	</ul>
	<p>The site does not use advertising cookies or a third-party analytics tool.</p>

	<h2>Discord intents</h2>
	<p>The bot asks Discord for the Message Content privileged intent, plus guild, member, moderation, message, and reaction intents. Message content is used for automod, prefix commands, starboard, and delete or edit logs. Member data is used for joins, leaves, roles, timeouts, and the server application. The bot only uses that data on servers where it is installed.</p>

	<h2>What the bot stores</h2>
	<p>For each server that uses the bot, the operator's database can hold:</p>
	<ul>
		<li>The server id, name, and icon, and the settings staff save.</li>
		<li>Cases: the action, the member's id and name, the moderator's id and name, the reason, the time, and a duration when there is one.</li>
		<li>Announcements staff write, including who wrote them.</li>
		<li>Temporary role grants: who received the role, who granted it, why, and when it ends.</li>
		<li>Server application status for a member, such as pending, cleared, or removed, with the join time and the deadline. Written answers are not stored. An age number is not stored. A removal for being under 18 is a case with the reason "Under the age requirement".</li>
		<li>Confessions: the text, the author's Discord id and name, the reviewer, and whether it was approved or rejected.</li>
		<li>Custom commands staff save.</li>
	</ul>

	<h2>What stays in Discord</h2>
	<p>The bot can read message text in a server where it is installed. It uses that text for automod, prefix commands, starboard, and delete or edit logs. Those logs are posted to a channel the server chose. The message text is not copied into the database for that purpose.</p>
	<p>A confession file is read so the bot can post it in Discord. The file is not saved in the database. After it is posted, Discord hosts it. The public confession does not include the author's name. Moderators can still see the author on the review post.</p>

	<h2>Who can see it</h2>
	<p>The dashboard shows a server to people Discord says can manage that server, and to people who hold a dashboard role the server saved. Confession authors are visible to moderators. Everyone who can read the confessions channel can see an approved confession, without the author's name. Discord receives what the bot posts, because the bot runs inside Discord.</p>

	<h2>How it is used</h2>
	<p>The information is used to run the dashboard, to carry out the moderation and server tools staff turned on, and to keep a record of those actions. It is not sold. It is not used for advertising. It is shared with Discord, with the staff of the server it belongs to, and when the law requires it.</p>

	<h2>Third parties</h2>
	<p>Discord provides account sign-in, the bot gateway, and message delivery. The operator runs the SquirrelBot process and a Postgres database that holds the records above. Hosting for that process and database is under the operator's control. No advertising network receives this information.</p>

	<h2>How long it is kept</h2>
	<p>A session lasts 14 days, or until you sign out. The sign-in check cookie lasts about 10 minutes. Server records stay so the bot can keep doing the job that server asked for. Removing the bot stops new actions and does not by itself delete rows already stored. Ask us to delete a server's records, or your session, and we will.</p>

	<h2>Your choices</h2>
	<p>Sign out to end your dashboard session. Ask a server's staff about a moderation case on that server. Open a request on <a href="https://github.com/BoxingOctopus/squirrelbot">github.com/BoxingOctopus/squirrelbot</a> when you want the operator to delete a session or a server's stored records.</p>

	<h2>Children</h2>
	<p>The dashboard is for people who manage a Discord server. A server application can remove a member who answers that they are under 18. SquirrelBot does not build a profile from that answer.</p>

	<h2>Contact</h2>
	<p>Questions about this policy, or a request to see or delete information, can go to <a href="https://github.com/BoxingOctopus/squirrelbot">github.com/BoxingOctopus/squirrelbot</a>. For a moderation case inside a server, the moderators of that server can see the case in the dashboard.</p>
</Legal>
