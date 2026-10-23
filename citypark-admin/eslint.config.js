import js from '@eslint/js'
import reactHooks from 'eslint-plugin-react-hooks'
import globals from 'globals'
import tseslint from 'typescript-eslint'

export default tseslint.config(
	{
		ignores: [
			'dist',
			'node_modules',
			'playwright-report',
			'test-results',
			'src/routeTree.gen.ts',
			'src/shared/api/schema.d.ts',
		],
	},
	js.configs.recommended,
	...tseslint.configs.recommended,
	reactHooks.configs.flat.recommended,
	{ languageOptions: { globals: globals.browser } }
)
