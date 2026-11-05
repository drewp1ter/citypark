import type { SiteSettings } from '@/shared/config'
import { formatHours, type WorkingHours } from '@/shared/lib'
import { AppLink, cardClass } from '@/shared/ui'

const questionCard = cardClass('surface', 'rounded-lg p-5 md:rounded-xl md:p-7')
const questionHeading = 'mb-2 text-lg font-bold md:text-xl'
const answer = 'text-text-muted leading-relaxed'
const link = 'tap-area text-text font-semibold underline'

/** Short answers drawn from the existing contacts and linked public pages. */
export function ContactsQuestions({ settings, workingHours }: { settings: SiteSettings; workingHours: WorkingHours }) {
	return (
		<section
			id='questions'
			aria-labelledby='contacts-questions-heading'
			className='max-w-page mx-auto w-full px-3 pt-10 pb-6 md:px-6 md:pt-[104px]'
		>
			<h2 id='contacts-questions-heading' className='heading mb-5 text-[26px] md:mb-6 md:text-4xl'>
				Частые вопросы
			</h2>
			<div className='grid gap-2 md:grid-cols-2 md:gap-3'>
				<div className={questionCard}>
					<h3 className={questionHeading}>Где находится «Сити Парк»?</h3>
					<p className={answer}>{settings.address.full}</p>
				</div>
				<div className={questionCard}>
					<h3 className={questionHeading}>Есть ли парковка?</h3>
					<p className={answer}>Для гостей — бесплатная парковка у здания.</p>
				</div>
				<div className={questionCard}>
					<h3 className={questionHeading}>В какое время работают ресторан и доставка?</h3>
					<p className={answer}>
						{formatHours(workingHours)}. Телефон ресторана и доставки:{' '}
						<AppLink href={settings.phones.restaurant.href} className={`${link} whitespace-nowrap`}>
							{settings.phones.restaurant.display}
						</AppLink>
						.
					</p>
				</div>
				<div className={questionCard}>
					<h3 className={questionHeading}>Где посмотреть номера, цены и правила проживания?</h3>
					<p className={answer}>
						<AppLink href='/rooms' className={link}>
							Номера и цены
						</AppLink>{' '}
						·{' '}
						<AppLink href='/hotel' className={link}>
							Правила проживания
						</AppLink>
						. Телефон отеля:{' '}
						<AppLink href={settings.phones.hotel.href} className={`${link} whitespace-nowrap`}>
							{settings.phones.hotel.display}
						</AppLink>
						.
					</p>
				</div>
				<div className={questionCard}>
					<h3 className={questionHeading}>Где посмотреть меню доставки и цены на блюда?</h3>
					<p className={answer}>
						<AppLink href='/menu' className={link}>
							Меню доставки
						</AppLink>
						.
					</p>
				</div>
			</div>
		</section>
	)
}
