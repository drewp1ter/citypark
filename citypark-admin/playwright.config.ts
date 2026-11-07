import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
	testDir: './e2e',
	testIgnore: 'pwa.spec.ts',
	use: { baseURL: 'http://localhost:5173' },
	projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
	webServer: { command: 'npm run dev', url: 'http://localhost:5173', reuseExistingServer: true },
})
