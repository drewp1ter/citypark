import type { ReactNode } from 'react'
import { MobileQuickActions } from '@/widgets/mobile-quick-actions'
import { SiteFooter } from '@/widgets/site-footer'
import { SiteHeader } from '@/widgets/site-header'

type SiteLayoutProps = {
	children: ReactNode
	/** The mobile "Позвонить / Забронировать" bar. The design has it on home, rooms and restaurant only. */
	quickActions?: boolean
}

/**
 * Default page frame: sticky header, content, footer, and (optionally) the mobile quick-actions bar after the
 * footer, so it stays pinned to the bottom of the screen for the whole page, as in the design.
 */
export function SiteLayout({ children, quickActions = true }: SiteLayoutProps) {
	return (
		<div className='flex min-h-dvh flex-col'>
			<SiteHeader />
			<div id='content' tabIndex={-1} className='flex-1 outline-none'>
				{children}
			</div>
			<SiteFooter />
			{quickActions && <MobileQuickActions />}
		</div>
	)
}
