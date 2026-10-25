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
		await expect(page.getByText('Moderation Without Enshittification.')).toBeVisible();
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
		await expect(page.getByRole('link', { name: 'Canopy' })).toBeVisible();
		await expect(page.getByText('Acorn')).toBeVisible();
		await page.getByRole('link', { name: 'Canopy' }).click();
		await expect(page.getByRole('heading', { name: 'On duty' })).toBeVisible();

		const sections: [string, string][] = [
			['Profile', 'Profile'],
			['Announcements', 'Announcements'],
			['Admin Firecall Access', 'Admin Firecall Access'],
			['Server Application', 'Server application'],
			['Cases', 'Cases'],
			['Automod', 'Automod'],
			['Logging', 'Logging'],
			['Welcome', 'Welcome'],
			['Reaction Roles', 'Reaction roles'],
			['Starboard', 'Starboard'],
			['Commands', 'Commands']
		];
		const nav = page.getByRole('navigation', { name: 'Server sections' });
		for (const [link, heading] of sections) {
			await nav.getByRole('link', { name: link }).click();
			await expect(page.getByRole('heading', { name: heading, exact: true })).toBeVisible();
		}
	});

	test('saves the server profile', async ({ page }) => {
		await openGuild(page);
		await page.getByRole('link', { name: 'Profile' }).click();
		await expect(page.getByText('squirrelbot', { exact: true })).toBeVisible();
		await page.getByLabel('Display name').fill('Canopy Guide');
		await page.getByRole('button', { name: 'Save profile' }).click();
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
		await page.getByLabel('Custom command prefix').fill('?');
		await page.getByRole('button', { name: 'Save welcome' }).click();
		await expect(page.getByText('Saved.')).toBeVisible();
		await page.reload();
		await expect(page.getByLabel('Custom command prefix')).toHaveValue('?');
		await expect(page.getByLabel('Welcome message')).toHaveValue('Hello {user}');
	});

	test('saves automod', async ({ page }) => {
		await openGuild(page);
		await page.getByRole('link', { name: 'Automod' }).click();
		await page.getByLabel('Automod is on').check();
		await page.getByLabel('Blocked words').fill('nope');
		await page.getByLabel('After deleting').selectOption({ label: 'Delete and warn' });
		await page.getByRole('button', { name: 'Save automod' }).click();
		await expect(page.getByText('Automod saved.')).toBeVisible();
	});

	test('creates a log channel and saves logging', async ({ page }) => {
		await openGuild(page);
		await page.getByRole('link', { name: 'Logging' }).click();
		await page.getByPlaceholder('Channel name').fill('mod-logs');
		await page.getByRole('button', { name: 'Create channel' }).click();
		await expect(page.getByText(/Created #mod-logs/)).toBeVisible();
		await page.getByRole('button', { name: 'Save logging' }).click();
		await expect(page.getByText('Saved.')).toBeVisible();
	});

	test('saves the starboard', async ({ page }) => {
		await openGuild(page);
		await page.getByRole('link', { name: 'Starboard' }).click();
		await page.getByLabel('Starboard channel').selectOption({ label: '#announcements' });
		await page.getByLabel('Stars required').fill('4');
		await page.getByRole('button', { name: 'Save starboard' }).click();
		await expect(page.getByText('Saved.')).toBeVisible();
	});

	test('saves a command script', async ({ page }) => {
		await openGuild(page);
		await page.getByRole('link', { name: 'Commands' }).click();
		await page.getByLabel('Name', { exact: true }).fill('hello');
		await page.getByLabel('Script').fill('return "hello"');
		await page.getByLabel('Roles this command may change').selectOption({ label: 'Members' });
		await page.getByRole('button', { name: 'Save command' }).click();
		await expect(page.getByText('Saved.')).toBeVisible();
		await expect(page.getByText('?hello')).toBeVisible();
	});

	test('sends an announcement and fills a template', async ({ page }) => {
		await openGuild(page);
		await page.getByRole('link', { name: 'Announcements' }).click();
		const publish = page.locator('form').first();
		await publish.getByLabel('Title').fill('Canopy night');
		await publish.getByLabel('Body').fill('The canopy is open.');
		await publish.getByRole('button', { name: 'Send now' }).click();
		await expect(page.getByText('Sent.')).toBeVisible();
		await expect(page.getByText('Canopy night')).toBeVisible();

		const template = page.locator('form').nth(1);
		await template.getByLabel('Name', { exact: true }).fill('Outage');
		await template.getByLabel('Title').fill('{{ server }} downtime');
		await template.getByLabel('Body').fill('{{ server }} pauses at {{ when }}. Posted by {{ author }}.');
		await template.getByRole('button', { name: 'Save template' }).click();
		await expect(page.getByText('Template saved.')).toBeVisible();
		await page.getByLabel('Values for the next use').fill('when=Tonight');
		await page.getByRole('button', { name: 'Use' }).click();
		await expect(page.getByText('Filled from the template. Send it when it looks right.')).toBeVisible();
		await expect(publish.getByLabel('Title')).toHaveValue('Canopy downtime');
		await expect(publish.getByLabel('Body')).toHaveValue('Canopy pauses at Tonight. Posted by Acorn.');
	});

	test('grants and ends admin firecall access', async ({ page }) => {
		await openGuild(page);
		await page.getByRole('navigation', { name: 'Server sections' }).getByRole('link', { name: 'Admin Firecall Access' }).click();
		const allow = page.locator('section').filter({ has: page.getByRole('heading', { name: 'Allowlist' }) });
		await allow.getByLabel('Role').selectOption({ label: 'Mods' });
		await allow.getByRole('button', { name: 'Add role' }).click();
		await allow.getByRole('button', { name: 'Save allowlist' }).click();
		await expect(page.getByText('Allowlist saved.')).toBeVisible();

		const grant = page.locator('form').filter({ has: page.getByRole('heading', { name: 'Grant' }) });
		await page.getByRole('combobox', { name: 'Find a member' }).click();
		await page.getByRole('option', { name: 'Fern' }).click();
		await grant.getByLabel('Role').selectOption({ label: 'Mods' });
		await grant.getByLabel('Reason').fill('Covering the night shift');
		await grant.getByRole('button', { name: 'Grant role' }).click();
		await expect(page.getByText('Granted to Fern.')).toBeVisible();
		await page.getByRole('button', { name: 'End now' }).click();
		await expect(page.getByRole('button', { name: 'End now' })).toHaveCount(0);
		await expect(page.getByText('revoked')).toBeVisible();
	});

	test('saves the application and posts the message', async ({ page }) => {
		await openGuild(page);
		await page.getByRole('link', { name: 'Server Application' }).click();
		await page.getByLabel('Kick people who miss the deadline').check();
		await page.getByRole('button', { name: 'Add question' }).click();
		await page.getByPlaceholder('Question').fill('Where did you find us?');
		await page.getByRole('button', { name: 'Save application' }).click();
		await expect(page.getByText('Saved.')).toBeVisible();
		await page.getByRole('button', { name: 'Post message' }).click();
		await expect(page.getByText('Message posted.')).toBeVisible();
	});

	test('adds a reaction role', async ({ page }) => {
		await openGuild(page);
		await page.getByRole('link', { name: 'Reaction Roles' }).click();
		await page.getByPlaceholder('Channel name').fill('pick-roles');
		await page.getByRole('button', { name: 'Create channel' }).click();
		await expect(page.getByText(/Created #pick-roles/)).toBeVisible();
		await page.getByRole('button', { name: 'Save channel' }).click();
		await expect(page.getByText('Roles channel saved.')).toBeVisible();
		await page.getByPlaceholder('Role name').fill('Helpers');
		await page.getByRole('button', { name: 'Create role' }).click();
		await expect(page.getByText('Created Helpers.')).toBeVisible();
		await page.getByLabel('Emoji').fill('🔥');
		await page.getByRole('button', { name: 'Add reaction role' }).click();
		await expect(page.getByText('Saved. The bot posted the reaction.')).toBeVisible();
		await expect(page.getByText(/🔥/)).toBeVisible();
	});

	test('shows the firecall case', async ({ page }) => {
		await openGuild(page);
		await page.getByRole('link', { name: 'Cases' }).click();
		await expect(page.getByText('#1 escalate')).toBeVisible();
		await expect(page.getByText('#2 deescalate')).toBeVisible();
		await expect(page.getByText('Fern').first()).toBeVisible();
	});

	test('signs out', async ({ page }) => {
		await page.goto('/app');
		await page.getByRole('button', { name: 'Sign out' }).click();
		await expect(page).toHaveURL(/\/$/);
		await expect(page.getByRole('link', { name: 'Continue with Discord' })).toBeVisible();
	});
});

async function openGuild(page: import('@playwright/test').Page) {
	await page.goto('/app');
	await page.getByRole('link', { name: 'Canopy' }).click();
	await expect(page.getByRole('heading', { name: 'On duty' })).toBeVisible();
}
