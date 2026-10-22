'use client'

import { useSyncExternalStore } from 'react'
import { cn } from '@/shared/lib'

const subscribe = (callback: () => void) => {
	window.addEventListener('hashchange', callback)
	return () => window.removeEventListener('hashchange', callback)
}

const tabs = [
	{ hash: '#news', label: { mobile: 'Новости', desktop: 'Новости и акции' } },
	{ hash: '#blog', label: { mobile: 'Блог шефа', desktop: 'Блог шеф-повара' } },
] as const

/**
 * Jumps between the news list and the chef's blog. Both sections stay on the page, so these are links to
 * #news and #blog, not real tabs; the selected one follows the address (#blog selects the blog).
 */
export function SectionTabs() {
	const hash = useSyncExternalStore(
		subscribe,
		() => window.location.hash,
		() => ''
	)
	const current = hash === '#blog' ? '#blog' : '#news'

	return (
		<nav
			aria-label='Раздел'
			className='rounded-pill border-border bg-surface grid grid-cols-2 border p-1 max-md:w-full md:inline-flex'
		>
			{tabs.map((tab) => (
				<a
					key={tab.hash}
					href={tab.hash}
					aria-current={current === tab.hash ? 'true' : undefined}
					className={cn(
						'rounded-pill inline-flex min-h-11 items-center justify-center px-5',
						current === tab.hash ? 'bg-inverse-bg text-inverse-text font-semibold' : 'text-text'
					)}
				>
					<span className='md:hidden'>{tab.label.mobile}</span>
					<span className='max-md:hidden'>{tab.label.desktop}</span>
				</a>
			))}
		</nav>
	)
}
