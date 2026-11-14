import { defineConfig, devices } from '@playwright/test';

const api = 'http://127.0.0.1:18080';
const dashboard = 'http://127.0.0.1:4173';

export default defineConfig({
	testDir: './e2e',
	fullyParallel: false,
	workers: 1,
	timeout: 60_000,
	expect: { timeout: 15_000 },
	use: {
		baseURL: dashboard,
		trace: 'on-first-retry'
	},
	webServer: [
		{
			command: 'go run -tags e2e ./e2e/cmd/stack',
			cwd: '..',
			url: `${api}/healthz`,
			reuseExistingServer: false,
			timeout: 120_000,
			env: {
				E2E_HTTP_ADDR: '127.0.0.1:18080',
				E2E_PUBLIC_URL: dashboard
			}
		},
		{
			command: 'npm run dev -- --host 127.0.0.1 --port 4173 --strictPort',
			url: dashboard,
			reuseExistingServer: false,
			timeout: 120_000,
			env: {
				SQUIRRELBOT_API: api
			}
		}
	],
	projects: [
		{
			name: 'chromium',
			use: { ...devices['Desktop Chrome'] }
		}
	]
});
