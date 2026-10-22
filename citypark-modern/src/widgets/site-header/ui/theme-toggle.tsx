'use client'

import { useTheme } from 'next-themes'
import { IconButton } from '@/shared/ui'

const iconProps = {
	width: 18,
	height: 18,
	viewBox: '0 0 24 24',
	fill: 'none',
	stroke: 'currentColor',
	strokeWidth: 1.8,
	strokeLinecap: 'round',
	strokeLinejoin: 'round',
	'aria-hidden': true,
} as const

/**
 * Sun/moon button. Both icons are rendered and CSS (the `dark:` variant on
 * [data-theme]) shows the right one, so there is no hydration mismatch.
 */
export function ThemeToggle() {
	const { resolvedTheme, setTheme } = useTheme()

	return (
		<IconButton aria-label='Переключить тему' onClick={() => setTheme(resolvedTheme === 'dark' ? 'light' : 'dark')}>
			<svg {...iconProps} className='block dark:hidden'>
				<path d='M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z' />
			</svg>
			<svg {...iconProps} className='hidden dark:block'>
				<circle cx='12' cy='12' r='4' />
				<path d='M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4' />
			</svg>
		</IconButton>
	)
}
