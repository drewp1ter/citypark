import { site } from '@/shared/config'
import { nb, plural } from '@/shared/lib'
import { AppLink } from '@/shared/ui'

const TOTAL_ROOMS = 12

/** Free rooms today: a lime tile with the number and a short caption. Zero shows a phone prompt instead. */
export function FreeRooms({ count }: { count: number }) {
	const { hotel } = site.phones

	if (count === 0) {
		return (
			<p className='text-text-muted mt-3 md:mt-4'>
				{nb('Сегодня свободных номеров нет.')} Позвоните: <AppLink href={hotel.href}>{hotel.display}</AppLink>
			</p>
		)
	}

	const phrase = plural(count, ['свободный номер', 'свободных номера', 'свободных номеров'])
	const noun = plural(count, ['номер', 'номера', 'номеров'])
	const verb = plural(count, ['свободен', 'свободно', 'свободно'])

	return (
		<p className='mt-3 flex items-center gap-3 md:mt-4 md:gap-4'>
			<span className='sr-only'>
				Сегодня {verb} {count} {noun} из {TOTAL_ROOMS}
			</span>
			<span
				aria-hidden
				className='bg-accent font-display text-on-accent flex size-16 shrink-0 items-center justify-center rounded-xs text-[40px] font-medium tracking-[-0.03em] tabular-nums md:size-[88px] md:rounded-sm md:text-[56px]'
			>
				{count}
			</span>
			<span aria-hidden className='flex flex-col'>
				<span className='text-text text-lg font-bold md:text-[22px]'>{phrase}</span>
				<span className='text-text-muted text-sm'>на сегодня из {TOTAL_ROOMS}</span>
			</span>
		</p>
	)
}
