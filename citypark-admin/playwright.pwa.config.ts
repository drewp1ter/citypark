import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
	testDir: './e2e',
	testMatch: 'pwa.spec.ts',
	use: { baseURL: 'http://localhost:4173' },
	projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
	webServer: { command: 'npm run preview -- --host 127.0.0.1 --port 4173 --strictPort', url: 'http://localhost:4173' },
})
