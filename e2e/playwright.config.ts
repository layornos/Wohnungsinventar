import { defineConfig } from '@playwright/test';

// Browser tests against the built server (bin/inventar with the embedded client). global-setup.ts builds it once;
// every worker then starts its own server with an empty data directory and a test user (fixtures.ts), so tests of
// different workers never share data, e.g. the single active move.
export default defineConfig({
	testDir: './tests',
	globalSetup: './global-setup.ts',
	fullyParallel: false,
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 1 : 0,
	reporter: [['list'], ['html', { open: 'never' }]],
	timeout: 30_000,
	expect: { timeout: 5_000 },
	use: {
		browserName: 'chromium',
		locale: 'de-DE',
		trace: 'retain-on-failure',
		screenshot: 'only-on-failure'
	},
	projects: [
		{
			name: 'phone',
			use: { viewport: { width: 390, height: 844 }, deviceScaleFactor: 3, isMobile: true, hasTouch: true }
		},
		{
			name: 'desktop',
			use: { viewport: { width: 1280, height: 800 } }
		}
	]
});
