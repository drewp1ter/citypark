import type { PageContent } from '@/entities/content'
import { getDeliveryInfo, type HomepageMenuGroup } from '@/entities/dish'
import { getWorkingHours } from '@/shared/api'
import { formatHours, formatPrice } from '@/shared/lib'
import { AppLink, cardClass } from '@/shared/ui'
import { DeliveryShowcase } from './delivery-showcase'

export async function DeliverySection({ content, groups }: { content: PageContent; groups: HomepageMenuGroup[] }) {
	const [hours, delivery] = await Promise.all([getWorkingHours(), getDeliveryInfo()])
	return (
		<DeliveryShowcase title={content.text('home.delivery.heading')} groups={groups}>
			<AppLink
				href='/menu'
				className={cardClass('inverse', 'hidden min-h-[270px] justify-between gap-4 rounded-lg p-7 md:flex')}
			>
				<span className='font-display text-[22px] leading-[1.15] font-medium tracking-[-0.02em]'>
					{content.text('home.delivery.all-menu-title')}
				</span>
				<span className='text-inverse-muted text-[15px]'>
					{delivery.city && delivery.city.freeFrom > 0 && (
						<>
							{content.text('home.delivery.all-menu-note')} {formatPrice(delivery.city.freeFrom)} ·{' '}
						</>
					)}
					{formatHours(hours)} · {delivery.outskirtsFee}
				</span>
				<span
					aria-hidden='true'
					className='bg-accent text-on-accent inline-flex size-11 items-center justify-center self-start rounded-full text-xl'
				>
					↗
				</span>
			</AppLink>
		</DeliveryShowcase>
	)
}
