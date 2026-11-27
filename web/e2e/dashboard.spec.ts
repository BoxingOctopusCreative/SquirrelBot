import { expect, test } from '@playwright/test';

const session = {
	name: 'sb_session',
	value: 'e2e-session',
	url: 'http://127.0.0.1:4173'
};

test.describe('signed out', () => {
	test('landing page offers Discord sign-in', async ({ page }) => {
		await page.goto('/');
		await expect(page.getByRole('link', { name: 'Continue with Discord' })).toHaveAttribute('href', '/auth/discord');
		await expect(page.getByRole('link', { name: 'Add SquirrelBot to a Server' })).toHaveAttribute('href', '/auth/discord/invite');
		await expect(page.getByRole('link', { name: 'Terms of Service' })).toHaveAttribute('href', '/terms');
		await expect(page.getByRole('link', { name: 'Privacy Policy' })).toHaveAttribute('href', '/privacy');
		await expect(page.getByText('Sign in to open servers where SquirrelBot is already installed.')).toBeVisible();
		await expect(page.getByText('Moderation Without Enshittification.')).toBeVisible();
	});

	test('terms and privacy are public', async ({ page }) => {
		await page.goto('/terms');
		await expect(page.getByRole('heading', { name: 'Terms of Service' })).toBeVisible();
		await expect(page.getByRole('link', { name: 'Privacy Policy' })).toBeVisible();
		await page.goto('/privacy');
		await expect(page.getByRole('heading', { name: 'Privacy Policy' })).toBeVisible();
		await expect(page.getByText('sb_session')).toBeVisible();
		await expect(page.getByText('Message Content')).toBeVisible();
	});

	test('a refused sign-in shows the banner', async ({ page }) => {
		await page.goto('/?error=denied');
		await expect(page.getByText('Discord did not finish sign-in. Try again.')).toBeVisible();
	});

	test('the dashboard sends an anonymous visitor home', async ({ page }) => {
		await page.goto('/app');
		await expect(page).toHaveURL(/\/$/);
		await expect(page.getByRole('link', { name: 'Continue with Discord' })).toBeVisible();
	});
});

test.describe('dashboard', () => {
	test.describe.configure({ mode: 'serial' });

	test.beforeEach(async ({ context }) => {
		await context.addCookies([session]);
	});

	test('lists the server and opens every section', async ({ page }) => {
		await page.goto('/app');
		const addServer = page.getByRole('link', { name: 'Add a Server' });
		await expect(addServer).toHaveAttribute('href', '/auth/discord/invite');
		await expect(page.getByRole('link', { name: 'Canopy' })).toBeVisible();
		await expect(page.getByText('Acorn')).toBeVisible();
		const username = await page.getByText('Acorn', { exact: true }).boundingBox();
		const addBox = await addServer.boundingBox();
		const themeButton = page.getByRole('button', { name: /Use (Light|Dark) Mode/ });
		const theme = await themeButton.boundingBox();
		expect(username && addBox && theme).toBeTruthy();
		expect(username!.x).toBeLessThan(theme!.x);
		expect(theme!.x + theme!.width).toBeLessThanOrEqual(addBox!.x + 8);
		const before = await themeButton.evaluate((el) => ({
			bg: getComputedStyle(el).backgroundColor,
			sun: el.querySelector('circle') !== null
		}));
		await themeButton.click();
		const after = await themeButton.evaluate((el) => ({
			bg: getComputedStyle(el).backgroundColor,
			sun: el.querySelector('circle') !== null
		}));
		expect(after.bg).not.toBe(before.bg);
		expect(after.sun).toBe(!before.sun);
		await themeButton.click();
		await expect(page.getByText(/^Version \S+$/)).toBeVisible();
		await page.getByRole('link', { name: 'Canopy' }).click();
		await expect(page.getByRole('heading', { name: 'Overview' })).toBeVisible();

		const sections: [string, string][] = [
			['Dashboard Access', 'Dashboard Access'],
			['Profile', 'Profile'],
			['Announcements', 'Announcements'],
			['Admin Firecall Access', 'Admin Firecall Access'],
			['Server Application', 'Server Application'],
			['Moderation Log', 'Moderation Log'],
			['Automod', 'Automod'],
			['Logging', 'Logging'],
			['Welcome', 'Welcome'],
			['Reaction Roles', 'Reaction Roles'],
			['Starboard', 'Starboard'],
			['Levels', 'Levels'],
			['Confessions', 'Confessions'],
			['Support Tickets', 'Support Tickets'],
			['Commands', 'Commands']
		];
		const nav = page.getByRole('navigation', { name: 'Server sections' });
		const overview = await nav.getByRole('link', { name: 'Overview' }).boundingBox();
		const profile = await nav.getByRole('link', { name: 'Profile' }).boundingBox();
		expect(overview).toBeTruthy();
		expect(profile).toBeTruthy();
		expect(profile!.y - (overview!.y + overview!.height)).toBeLessThan(2);
		for (const [link, heading] of sections) {
			await nav.getByRole('link', { name: link }).click();
			await expect(page.getByRole('heading', { name: heading, exact: true })).toBeVisible();
		}
	});

	test('saves dashboard access roles', async ({ page }) => {
		await openGuild(page);
		await page.getByRole('navigation', { name: 'Server sections' }).getByRole('link', { name: 'Dashboard Access' }).click();
		await page.getByRole('combobox', { name: 'Role', exact: true }).selectOption({ label: 'Mods' });
		await page.getByRole('button', { name: 'Add Role' }).click();
		await page.getByRole('button', { name: 'Save Access' }).click();
		await expect(page.getByText('Saved.')).toBeVisible();
		await expect(page.locator('.list').getByText('Mods')).toBeVisible();
		await page.reload();
		await expect(page.locator('.list').getByText('Mods')).toBeVisible();
		await expect(page.getByLabel('Role name', { exact: true })).toHaveValue('Moderator');
		await page.getByLabel('Role name', { exact: true }).fill('Night mods');
		await page.getByRole('button', { name: 'Create Role' }).click();
		await expect(page.getByText('Created Night mods.')).toBeVisible();
		await expect(page.locator('.list').getByText('Night mods')).toBeVisible();
		await page.reload();
		await expect(page.locator('.list').getByText('Night mods')).toBeVisible();
		await expect(page.getByLabel('Role name', { exact: true })).toHaveValue('Moderator');
	});

	test('saves the server profile', async ({ page }) => {
		await openGuild(page);
		await page.getByRole('link', { name: 'Profile' }).click();
		await expect(page.getByText('squirrelbot', { exact: true })).toBeVisible();
		await page.getByLabel('Display name').fill('Canopy Guide');
		await page.getByRole('button', { name: 'Save Profile' }).click();
		await expect(page.getByText('Saved.')).toBeVisible();
		await expect(page.getByLabel('Display name')).toHaveValue('Canopy Guide');
	});

	test('saves welcome settings', async ({ page }) => {
		await openGuild(page);
		await page.getByRole('link', { name: 'Welcome' }).click();
		await page.getByLabel('Mod role').selectOption({ label: 'Mods' });
		await page.getByLabel('Admin role').selectOption({ label: 'Admins' });
		await page.getByLabel('Welcome channel').selectOption({ label: '#general' });
		await page.getByLabel('Welcome message').fill('Hello {user}');
		await page.getByLabel('Leave channel').selectOption({ label: '#announcements' });
		await page.getByLabel('Leave message').fill('Goodbye {username}');
		await page.getByLabel('Custom command prefix').fill('?');
		await expect(page.getByLabel('Welcome on')).toBeChecked();
		const saveWelcome = page.getByRole('button', { name: 'Save Welcome' });
		const addServer = page.getByRole('link', { name: 'Add a Server' });
		const saveColor = await saveWelcome.evaluate((el) => getComputedStyle(el).backgroundColor);
		await expect(addServer).toHaveCSS('background-color', saveColor);
		await saveWelcome.click();
		await expect(page.getByText('Saved.')).toBeVisible();
		await page.reload();
		await expect(page.getByLabel('Custom command prefix')).toHaveValue('?');
		await expect(page.getByLabel('Welcome message')).toHaveValue('Hello {user}');
		await expect(page.getByLabel('Leave message')).toHaveValue('Goodbye {username}');
	});

	test('saves automod', async ({ page }) => {
		await openGuild(page);
		await page.getByRole('link', { name: 'Automod' }).click();
		await page.getByLabel('Automod is on').check();
		await page.getByLabel('Blocked words').fill('nope');
		await page.getByLabel('After deleting').selectOption({ label: 'Delete and warn' });
		await page.getByRole('button', { name: 'Save Automod' }).click();
		await expect(page.getByText('Automod saved.')).toBeVisible();
	});

	test('creates a log channel and saves logging', async ({ page }) => {
		await openGuild(page);
		await page.getByRole('link', { name: 'Logging' }).click();
		await page.getByPlaceholder('Channel name').fill('mod-logs');
		await page.getByRole('button', { name: 'Create Channel' }).click();
		await expect(page.getByText(/Created #mod-logs/)).toBeVisible();
		await expect(page.getByLabel('Logging on')).toBeChecked();
		await page.getByRole('button', { name: 'Save Logging' }).click();
		await expect(page.getByText('Saved.')).toBeVisible();
		await page.getByLabel('Logging on').uncheck();
		await Promise.all([
			page.waitForResponse((res) => res.url().includes('/logging') && res.request().method() === 'PUT' && res.ok()),
			page.getByRole('button', { name: 'Save Logging' }).click()
		]);
		await page.reload();
		await expect(page.getByLabel('Logging on')).not.toBeChecked();
		await expect(page.getByText('Turn this on to record message and member events.')).toBeVisible();
		await page.getByLabel('Logging on').check();
		await Promise.all([
			page.waitForResponse((res) => res.url().includes('/logging') && res.request().method() === 'PUT' && res.ok()),
			page.getByRole('button', { name: 'Save Logging' }).click()
		]);
	});

	test('saves confession channels', async ({ page }) => {
		await openGuild(page);
		await page.getByRole('link', { name: 'Confessions' }).click();
		const names = page.getByPlaceholder('Channel name');
		await names.nth(0).fill('confession-review');
		await page.getByRole('button', { name: 'Create Channel' }).nth(0).click();
		await expect(page.getByText(/Created #confession-review/)).toBeVisible();
		await names.nth(1).fill('confessions');
		await page.getByRole('button', { name: 'Create Channel' }).nth(1).click();
		await expect(page.getByText(/Created #confessions/)).toBeVisible();
		await expect(page.getByLabel('Confessions on')).toBeChecked();
		await page.getByRole('button', { name: 'Save Confessions' }).click();
		await expect(page.getByText('Saved.')).toBeVisible();
		await page.reload();
		await expect(page.getByLabel('Review channel')).toHaveValue(/.+/);
		await expect(page.getByLabel('Confessions channel')).not.toHaveValue('');
	});

	test('saves support tickets and posts the panel', async ({ page }) => {
		await openGuild(page);
		await page.getByRole('navigation', { name: 'Server sections' }).getByRole('link', { name: 'Support Tickets' }).click();
		const panel = page.getByRole('combobox', { name: /Panel channel/ });
		await panel.selectOption({ label: '#general' });
		await expect(panel.locator('option:checked')).toHaveText('#general');
		await page.getByLabel('Category name').fill('tickets');
		await page.getByRole('button', { name: 'Create Category' }).click();
		await expect(page.getByText(/Created the tickets category/)).toBeVisible();
		await page.getByLabel('Transcript channel name').fill('ticket-logs');
		await page.getByRole('button', { name: 'Create Channel' }).nth(1).click();
		await expect(page.getByText(/Created #ticket-logs/)).toBeVisible();
		await expect(panel.locator('option:checked')).toHaveText('#general');
		await page.getByLabel('Support role').selectOption({ label: 'Mods' });
		await expect(page.getByLabel('Support tickets on')).toBeChecked();
		await page.getByRole('button', { name: 'Save and Post the Panel' }).click();
		await expect(page.getByText('Message posted.')).toBeVisible();
		await page.reload();
		await expect(page.getByRole('heading', { name: 'Support Tickets', exact: true })).toBeVisible();
		await expect(page.getByText(/deletes the ticket channel ten seconds later/)).toBeVisible();
		await expect(page.getByRole('combobox', { name: /Panel channel/ }).locator('option:checked')).toHaveText('#general');
		await expect(page.getByRole('combobox', { name: /Ticket category/ })).not.toHaveValue('');
		await expect(page.getByRole('combobox', { name: /Transcript channel/ })).not.toHaveValue('');
		await expect(page.getByRole('combobox', { name: /Support role/ })).toHaveValue(/.+/);
		await expect(page.getByRole('heading', { name: 'Open tickets' })).toBeVisible();
		await expect(page.getByText('No open tickets.')).toBeVisible();
	});

	test('saves the starboard', async ({ page }) => {
		await openGuild(page);
		await page.getByRole('link', { name: 'Starboard' }).click();
		await page.getByLabel('Starboard channel').selectOption({ label: '#announcements' });
		await page.getByLabel('Stars required').fill('4');
		await page.getByRole('button', { name: 'Save Starboard' }).click();
		await expect(page.getByText('Saved.')).toBeVisible();
	});

	test('saves levels', async ({ page }) => {
		await openGuild(page);
		await page.getByRole('link', { name: 'Levels' }).click();
		await expect(page.getByRole('heading', { name: 'Levels', exact: true })).toBeVisible();
		await page.getByRole('checkbox', { name: 'Levels on' }).check();
		await page.getByRole('button', { name: 'Save Levels' }).click();
		await expect(page.getByText('Saved.')).toBeVisible();
	});

	test('saves a command script', async ({ page }) => {
		await openGuild(page);
		await page.getByRole('link', { name: 'Commands' }).click();
		await page.getByLabel('Name', { exact: true }).fill('hello');
		await page.getByLabel('Script').fill('return "hello"');
		await page.getByLabel('Roles this command may change').selectOption({ label: 'Members' });
		await page.getByRole('button', { name: 'Save Command' }).click();
		await expect(page.getByText('Saved.')).toBeVisible();
		await expect(page.getByText('?hello')).toBeVisible();
	});

	test('sends an announcement and fills a template', async ({ page }) => {
		await openGuild(page);
		await page.getByRole('link', { name: 'Announcements' }).click();
		await expect(page.getByLabel('Announcements on')).toBeChecked();
		await page.getByLabel('Announcements on').uncheck();
		await page.getByRole('button', { name: 'Save', exact: true }).click();
		await expect(page.getByText('Saved. Announcements updated.')).toBeVisible();
		await page.reload();
		await expect(page.getByLabel('Announcements on')).not.toBeChecked();
		await page.getByLabel('Announcements on').check();
		await page.getByRole('button', { name: 'Save', exact: true }).click();
		await expect(page.getByText('Saved. Announcements updated.')).toBeVisible();
		const publish = page.locator('form').first();
		await publish.getByLabel('Title').fill('Canopy night');
		await publish.getByLabel('Body').fill('The canopy is open.');
		await publish.getByRole('button', { name: 'Send Now' }).click();
		await expect(page.getByText('Sent.')).toBeVisible();
		await expect(page.getByText('Canopy night')).toBeVisible();

		const template = page.locator('form').nth(1);
		await template.getByLabel('Name', { exact: true }).fill('Outage');
		await template.getByLabel('Title').fill('{{ server }} downtime');
		await template.getByLabel('Body').fill('{{ server }} pauses at {{ when }}. Posted by {{ author }}.');
		await template.getByRole('button', { name: 'Save Template' }).click();
		await expect(page.getByText('Template saved.')).toBeVisible();
		await page.getByLabel('Values for the next use').fill('when=Tonight');
		await page.getByRole('button', { name: 'Use', exact: true }).click();
		await expect(page.getByText('Filled from the template. Send it when it looks right.')).toBeVisible();
		await expect(publish.getByLabel('Title')).toHaveValue('Canopy downtime');
		await expect(publish.getByLabel('Body')).toHaveValue('Canopy pauses at Tonight. Posted by Acorn.');
	});

	test('grants and ends admin firecall access', async ({ page }) => {
		await openGuild(page);
		await page.getByRole('navigation', { name: 'Server sections' }).getByRole('link', { name: 'Admin Firecall Access' }).click();
		const allow = page.locator('section').filter({ has: page.getByRole('heading', { name: 'Allowlist' }) });
		await expect(allow.getByLabel('Admin firecall on')).toBeChecked();
		await allow.getByLabel('Role').selectOption({ label: 'Mods' });
		await allow.getByRole('button', { name: 'Add Role' }).click();
		await allow.getByRole('button', { name: 'Save Allowlist' }).click();
		await expect(page.getByText('Allowlist saved.')).toBeVisible();

		const grant = page.locator('form').filter({ has: page.getByRole('heading', { name: 'Grant' }) });
		await page.getByRole('combobox', { name: 'Find a member' }).click();
		await page.getByRole('option', { name: 'Fern' }).click();
		await grant.getByLabel('Role').selectOption({ label: 'Mods' });
		await grant.getByLabel('Reason').fill('Covering the night shift');
		await grant.getByRole('button', { name: 'Grant Role' }).click();
		await expect(page.getByText('Granted to Fern.')).toBeVisible();
		await page.getByRole('button', { name: 'End Now' }).click();
		await expect(page.getByRole('button', { name: 'End Now' })).toHaveCount(0);
		await expect(page.getByText('revoked')).toBeVisible();
	});

	test('saves the application and posts the message', async ({ page }) => {
		await openGuild(page);
		await page.getByRole('link', { name: 'Server Application' }).click();
		await expect(page.getByLabel('Server application on')).toBeChecked();
		await page.getByLabel('Kick people who miss the deadline').check();
		await page.getByRole('button', { name: 'Add Question' }).click();
		await page.getByPlaceholder('Question').fill('Where did you find us?');
		await page.getByRole('button', { name: 'Save Application' }).click();
		await expect(page.getByText('Saved.')).toBeVisible();
		await page.getByRole('button', { name: 'Post Message' }).click();
		await expect(page.getByText('Message posted.')).toBeVisible();
	});

	test('adds a reaction role', async ({ page }) => {
		await openGuild(page);
		await page.getByRole('link', { name: 'Reaction Roles' }).click();
		await expect(page.getByLabel('Reaction roles on')).toBeChecked();
		await page.getByLabel('Reaction roles on').uncheck();
		await page.getByRole('button', { name: 'Save', exact: true }).click();
		await expect(page.getByText('Saved. Reaction roles updated.')).toBeVisible();
		await page.reload();
		await expect(page.getByLabel('Reaction roles on')).not.toBeChecked();
		await page.getByLabel('Reaction roles on').check();
		await page.getByRole('button', { name: 'Save', exact: true }).click();
		await expect(page.getByText('Saved. Reaction roles updated.')).toBeVisible();
		await page.getByPlaceholder('Channel name').fill('pick-roles');
		await page.getByRole('button', { name: 'Create Channel' }).click();
		await expect(page.getByText(/Created #pick-roles/)).toBeVisible();
		await page.getByRole('button', { name: 'Save Channel' }).click();
		await expect(page.getByText('Roles channel saved.')).toBeVisible();
		await page.getByPlaceholder('Role name').fill('Helpers');
		await page.getByRole('button', { name: 'Create Role' }).click();
		await expect(page.getByText('Created Helpers.')).toBeVisible();
		await page.getByLabel('Emoji').fill('🔥');
		await page.getByRole('button', { name: 'Add Reaction Role' }).click();
		await expect(page.getByText('Saved. The bot posted the reaction.')).toBeVisible();
		await expect(page.getByText(/🔥/)).toBeVisible();
	});

	test('shows the firecall case', async ({ page }) => {
		await openGuild(page);
		await page.getByRole('link', { name: 'Moderation Log' }).click();
		await expect(page.getByText('#1 escalate')).toBeVisible();
		await expect(page.getByText('#2 deescalate')).toBeVisible();
		await expect(page.getByText('Fern').first()).toBeVisible();
	});

	test('signs out', async ({ page }) => {
		await page.goto('/app');
		await page.getByRole('button', { name: 'Sign Out' }).click();
		await expect(page).toHaveURL(/\/$/);
		await expect(page.getByRole('link', { name: 'Continue with Discord' })).toBeVisible();
	});
});

async function openGuild(page: import('@playwright/test').Page) {
	await page.goto('/app');
	await page.getByRole('link', { name: 'Canopy' }).click();
	await expect(page.getByRole('heading', { name: 'Overview' })).toBeVisible();
}
