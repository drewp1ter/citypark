import { cardClass } from '@/shared/ui'
import { banquets } from '../model/content'

/** Banquets: one short lime card on mobile, three cards (wedding, anniversary, corporate) on desktop. */
export function Banquets() {
	return (
		<section className='max-w-page mx-auto w-full px-3 pt-2 md:px-6 md:pt-[104px]'>
			<h2 className='heading mb-7 text-[52px] max-md:hidden'>Банкеты</h2>

			<div className={cardClass('accent', 'gap-1.5 rounded-lg p-[22px] md:hidden')}>
				<h2 className='heading text-[26px] tracking-[-0.02em]'>Банкеты</h2>
				<p>{banquets.short}</p>
			</div>

			<div className='grid gap-3 max-md:hidden md:grid-cols-3'>
				{banquets.items.map((item) => (
					<div
						key={item.title}
						className={cardClass(
							'accent' in item ? 'accent' : 'surface',
							'min-h-[220px] justify-between rounded-xl p-8'
						)}
					>
						<span className='font-display text-[26px] font-medium'>{item.title}</span>
						<span className={'accent' in item ? undefined : 'text-text-muted'}>{item.text}</span>
					</div>
				))}
			</div>
		</section>
	)
}
