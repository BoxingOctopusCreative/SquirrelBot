# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: web/e2e/dashboard.spec.ts >> signed out >> landing page offers Discord sign-in
- Location: web/e2e/dashboard.spec.ts:10:2

# Error details

```
Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
Call log:
  - navigating to "/", waiting until "load"

```

# Test source

```ts
  1   | import { expect, test } from '@playwright/test';
  2   | 
  3   | const session = {
  4   | 	name: 'sb_session',
  5   | 	value: 'e2e-session',
  6   | 	url: 'http://127.0.0.1:4173'
  7   | };
  8   | 
  9   | test.describe('signed out', () => {
  10  | 	test('landing page offers Discord sign-in', async ({ page }) => {
> 11  | 		await page.goto('/');
      |              ^ Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
  12  | 		await expect(page.getByRole('link', { name: 'Continue with Discord' })).toHaveAttribute('href', '/auth/discord');
  13  | 		await expect(page.getByRole('link', { name: 'Add SquirrelBot to a Server' })).toHaveAttribute('href', '/auth/discord/invite');
  14  | 		await expect(page.getByRole('link', { name: 'Terms of Service' })).toHaveAttribute('href', '/terms');
  15  | 		await expect(page.getByRole('link', { name: 'Privacy Policy' })).toHaveAttribute('href', '/privacy');
  16  | 		await expect(page.getByText('Sign in to open servers where SquirrelBot is already installed.')).toBeVisible();
  17  | 		await expect(page.getByText('Moderation Without Enshittification.')).toBeVisible();
  18  | 	});
  19  | 
  20  | 	test('terms and privacy are public', async ({ page }) => {
  21  | 		await page.goto('/terms');
  22  | 		await expect(page.getByRole('heading', { name: 'Terms of Service' })).toBeVisible();
  23  | 		await expect(page.getByRole('link', { name: 'Privacy Policy' })).toBeVisible();
  24  | 		await page.goto('/privacy');
  25  | 		await expect(page.getByRole('heading', { name: 'Privacy Policy' })).toBeVisible();
  26  | 		await expect(page.getByText('sb_session')).toBeVisible();
  27  | 		await expect(page.getByText('Message Content')).toBeVisible();
  28  | 	});
  29  | 
  30  | 	test('a refused sign-in shows the banner', async ({ page }) => {
  31  | 		await page.goto('/?error=denied');
  32  | 		await expect(page.getByText('Discord did not finish sign-in. Try again.')).toBeVisible();
  33  | 	});
  34  | 
  35  | 	test('the dashboard sends an anonymous visitor home', async ({ page }) => {
  36  | 		await page.goto('/app');
  37  | 		await expect(page).toHaveURL(/\/$/);
  38  | 		await expect(page.getByRole('link', { name: 'Continue with Discord' })).toBeVisible();
  39  | 	});
  40  | });
  41  | 
  42  | test.describe('dashboard', () => {
  43  | 	test.describe.configure({ mode: 'serial' });
  44  | 
  45  | 	test.beforeEach(async ({ context }) => {
  46  | 		await context.addCookies([session]);
  47  | 	});
  48  | 
  49  | 	test('lists the server and opens every section', async ({ page }) => {
  50  | 		await page.goto('/app');
  51  | 		await expect(page.getByRole('link', { name: 'Add a Server' })).toHaveAttribute('href', '/auth/discord/invite');
  52  | 		await expect(page.getByRole('link', { name: 'Canopy' })).toBeVisible();
  53  | 		await expect(page.getByText('Acorn')).toBeVisible();
  54  | 		await expect(page.getByText(/^Version \S+$/)).toBeVisible();
  55  | 		await page.getByRole('link', { name: 'Canopy' }).click();
  56  | 		await expect(page.getByRole('heading', { name: 'Overview' })).toBeVisible();
  57  | 
  58  | 		const sections: [string, string][] = [
  59  | 			['Dashboard Access', 'Dashboard Access'],
  60  | 			['Profile', 'Profile'],
  61  | 			['Announcements', 'Announcements'],
  62  | 			['Admin Firecall Access', 'Admin Firecall Access'],
  63  | 			['Server Application', 'Server Application'],
  64  | 			['Moderation Log', 'Moderation Log'],
  65  | 			['Automod', 'Automod'],
  66  | 			['Logging', 'Logging'],
  67  | 			['Welcome', 'Welcome'],
  68  | 			['Reaction Roles', 'Reaction Roles'],
  69  | 			['Starboard', 'Starboard'],
  70  | 			['Levels', 'Levels'],
  71  | 			['Confessions', 'Confessions'],
  72  | 			['Support Tickets', 'Support Tickets'],
  73  | 			['Commands', 'Commands']
  74  | 		];
  75  | 		const nav = page.getByRole('navigation', { name: 'Server sections' });
  76  | 		const overview = await nav.getByRole('link', { name: 'Overview' }).boundingBox();
  77  | 		const profile = await nav.getByRole('link', { name: 'Profile' }).boundingBox();
  78  | 		expect(overview).toBeTruthy();
  79  | 		expect(profile).toBeTruthy();
  80  | 		expect(profile!.y - (overview!.y + overview!.height)).toBeLessThan(2);
  81  | 		for (const [link, heading] of sections) {
  82  | 			await nav.getByRole('link', { name: link }).click();
  83  | 			await expect(page.getByRole('heading', { name: heading, exact: true })).toBeVisible();
  84  | 		}
  85  | 	});
  86  | 
  87  | 	test('saves dashboard access roles', async ({ page }) => {
  88  | 		await openGuild(page);
  89  | 		await page.getByRole('navigation', { name: 'Server sections' }).getByRole('link', { name: 'Dashboard Access' }).click();
  90  | 		await page.getByRole('combobox', { name: 'Role', exact: true }).selectOption({ label: 'Mods' });
  91  | 		await page.getByRole('button', { name: 'Add Role' }).click();
  92  | 		await page.getByRole('button', { name: 'Save Access' }).click();
  93  | 		await expect(page.getByText('Saved.')).toBeVisible();
  94  | 		await expect(page.locator('.list').getByText('Mods')).toBeVisible();
  95  | 		await page.reload();
  96  | 		await expect(page.locator('.list').getByText('Mods')).toBeVisible();
  97  | 		await expect(page.getByLabel('Role name', { exact: true })).toHaveValue('Moderator');
  98  | 		await page.getByLabel('Role name', { exact: true }).fill('Night mods');
  99  | 		await page.getByRole('button', { name: 'Create Role' }).click();
  100 | 		await expect(page.getByText('Created Night mods.')).toBeVisible();
  101 | 		await expect(page.locator('.list').getByText('Night mods')).toBeVisible();
  102 | 		await page.reload();
  103 | 		await expect(page.locator('.list').getByText('Night mods')).toBeVisible();
  104 | 		await expect(page.getByLabel('Role name', { exact: true })).toHaveValue('Moderator');
  105 | 	});
  106 | 
  107 | 	test('saves the server profile', async ({ page }) => {
  108 | 		await openGuild(page);
  109 | 		await page.getByRole('link', { name: 'Profile' }).click();
  110 | 		await expect(page.getByText('squirrelbot', { exact: true })).toBeVisible();
  111 | 		await page.getByLabel('Display name').fill('Canopy Guide');
```