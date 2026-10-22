import { site } from '@/shared/config'
import { cn } from '@/shared/lib'
import { AppLink, Logo } from '@/shared/ui'

const { company, legal, social, year } = site

export function SiteFooter({ wide = false }: { wide?: boolean }) {
	return (
		<footer className={cn('mx-auto w-full px-3 pt-7 pb-8 md:px-6 md:pt-10', wide ? 'max-w-wide-page' : 'max-w-page')}>
			<Logo className='mb-4 h-[120px] md:mb-8 md:h-[clamp(180px,18vw,240px)]' />

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
