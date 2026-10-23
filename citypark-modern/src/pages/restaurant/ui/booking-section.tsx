import { TableBookingForm } from '@/features/book-table'
import { getSiteSettings } from '@/shared/api'
import { AppLink } from '@/shared/ui'

/** Dark booking block: heading and phone numbers on the left (desktop), the form on the right. */
export async function BookingSection() {
	const { phones } = await getSiteSettings()

	return (
		<section id='booking' className='max-w-page mx-auto w-full scroll-mt-24 px-3 pt-12 md:px-6 md:pt-[104px] md:pb-6'>
			<div className='bg-inverse-bg text-inverse-text flex flex-wrap gap-10 rounded-[30px] p-[22px] md:rounded-2xl md:p-12'>
				<div className='hidden min-w-0 flex-[1_1_340px] flex-col gap-4 md:flex'>
					<h2 className='heading text-[40px] leading-[1.05]'>Забронировать столик</h2>
					<p className='text-inverse-muted'>Перезвоним, чтобы подтвердить бронь.</p>
					<AppLink href={phones.restaurant.href} className='tap-area font-display text-[26px] font-medium'>
						{phones.restaurant.display}
					</AppLink>
					{phones.restaurant.alt && (
						<AppLink href={phones.restaurant.alt.href} className='tap-area text-inverse-muted text-lg'>
							{phones.restaurant.alt.display}
						</AppLink>
					)}
				</div>
				<TableBookingForm variant='dark' className='min-w-0 flex-[1_1_520px]' />
			</div>
		</section>
	)
}
