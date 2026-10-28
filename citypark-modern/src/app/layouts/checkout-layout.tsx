import type { ReactNode } from 'react'
import { getSiteSettings, getWorkingHours } from '@/shared/api'
import { site } from '@/shared/config'
import { formatHours } from '@/shared/lib'

/** Checkout frame: no site navigation, just a quiet footer (the page brings its own header). */
export async function CheckoutLayout({ children }: { children: ReactNode }) {
	const [{ company, address }, hours] = await Promise.all([getSiteSettings(), getWorkingHours()])

	return (
		<div className='flex min-h-dvh flex-col'>
			<div id='content' tabIndex={-1} className='flex-1 outline-none'>
				{children}
			</div>
			<footer className='max-w-narrow-page text-text-muted mx-auto hidden w-full flex-wrap justify-between gap-3 px-6 pb-8 text-sm md:flex'>
				<span>
					© {site.year} {company.name} · {address.street}
				</span>
				<span>Доставка {formatHours(hours)}</span>
			</footer>
		</div>
	)
}
