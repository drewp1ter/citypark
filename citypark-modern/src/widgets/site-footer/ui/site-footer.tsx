import { site } from '@/shared/config'
import { cn } from '@/shared/lib'
import { AppLink } from '@/shared/ui'

const { company, legal, social, year } = site

export function SiteFooter({ wide = false }: { wide?: boolean }) {
	return (
		<footer className={cn('mx-auto w-full px-3 pt-7 pb-8 md:px-6 md:pt-10', wide ? 'max-w-wide-page' : 'max-w-page')}>
			<p className='font-display mb-3 text-[54px] leading-[0.9] font-semibold tracking-[-0.05em] md:mb-6 md:text-[clamp(88px,12vw,var(--text-wordmark))]'>
				СИТИ
				<br className='md:hidden' /> ПАРК
			</p>

			{/* Mobile: short line, as in the design */}
			<p className='text-text-muted text-[13px] md:hidden'>
				© {year} {company.name} ·{' '}
				{legal.slice(0, 2).map(({ href, label }, i) => (
					<span key={href}>
						{i > 0 && ' · '}
						<AppLink href={href} className='tap-area underline'>
							{label}
						</AppLink>
					</span>
				))}
			</p>

			{/* Desktop */}
			<div className='text-text-muted hidden flex-wrap justify-between gap-x-6 gap-y-3 text-[15px] md:flex'>
				<span>
					© {year} {company.name} · ИНН {company.inn}
				</span>
				<nav aria-label='Документы и соцсети' className='flex flex-wrap gap-x-6 gap-y-2'>
					{[...legal, ...social].map(({ href, label }) => (
						<AppLink key={label} href={href} className='tap-area underline'>
							{label}
						</AppLink>
					))}
				</nav>
			</div>
		</footer>
	)
}
