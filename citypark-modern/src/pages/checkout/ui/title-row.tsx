import { cn } from '@/shared/lib'

const steps = ['Корзина', 'Доставка', 'Готово']

/** The h1 and the three steps. On mobile the title lives in the header, so the h1 is only for screen readers there. */
export function TitleRow({ step, title }: { step: 2 | 3; title: string }) {
	return (
		<div className='flex flex-wrap items-end justify-between gap-4 md:mb-7'>
			<h1 className='heading text-[52px] max-md:sr-only'>{title}</h1>
			<ol aria-label='Шаги' className='hidden gap-1.5 text-sm font-semibold md:flex'>
				{steps.map((label, index) => {
					const number = index + 1
					const done = number < step
					const current = number === step
					return (
						<li
							key={label}
							aria-current={current ? 'step' : undefined}
							className={cn(
								'rounded-pill px-3.5 py-1.5',
								current ? 'bg-inverse-bg text-inverse-text' : 'border-border bg-surface border',
								!done && !current && 'text-text-muted'
							)}
						>
							{number} {label}
							{done && ' ✓'}
						</li>
					)
				})}
			</ol>
		</div>
	)
}
