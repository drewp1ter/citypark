import { defineConfig } from 'steiger'
import fsd from '@feature-sliced/steiger-plugin'

export default defineConfig([
	...fsd.configs.recommended,
	{ ignores: ['**/node_modules/**', '**/.next/**'] },
	{
		// Owner-approved temporary exception for task 55; remove when address management
		// and checkout become independent consumers in tasks 56–57.
		files: ['./src/entities/address/**'],
		rules: { 'fsd/insignificant-slice': 'off' },
	},
])
