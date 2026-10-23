import { getSiteSettings } from '@/shared/api'
import { AppLink, Button, PinIcon } from '@/shared/ui'

/** Inverse block with address, phones and the map. Shorter on mobile: no e-mail, no map. */
export async function ContactsBlock() {
	const { address, phones, email } = await getSiteSettings()
	const phoneRows = [phones.restaurant, phones.hotel, phones.free]

	return (
		<div className='bg-inverse-bg text-inverse-text flex flex-col gap-4 rounded-xl p-6 md:flex-row md:flex-wrap md:gap-10 md:rounded-2xl md:p-12'>
			<div className='flex min-w-0 flex-col gap-4 md:flex-[1_1_420px] md:gap-6'>
				<h2 className='font-display text-[26px] leading-[1.1] font-medium tracking-[-0.02em] md:text-[44px] md:leading-[1.05] md:tracking-[-0.03em]'>
					{address.street}
					<span className='text-inverse-subtle hidden md:inline'>
						<br />
						{address.city}
					</span>
				</h2>

				<ul className='border-inverse-border flex flex-col border-t'>
					{phoneRows.map((phone) => (
						<li key={phone.href}>
							<AppLink
								href={phone.href}
								className='border-inverse-border flex justify-between gap-3 border-b py-3.5 md:py-4 md:text-lg'
							>
								<span className='text-inverse-muted'>
									<span className='md:hidden'>{phone.shortLabel}</span>
									<span className='hidden md:inline'>{phone.label}</span>
								</span>
								<span className='font-semibold'>{phone.display}</span>
							</AppLink>
						</li>
					))}
				</ul>

				<div className='flex flex-wrap gap-2'>
					<Button href='/contacts' size='lg' className='max-md:w-full'>
						Построить маршрут
					</Button>
					<Button href={`mailto:${email}`} variant='outline-inverse' size='lg' className='hidden md:inline-flex'>
						{email}
					</Button>
				</div>
			</div>

			{/* Placeholder until the Yandex Map is embedded. */}
			<div className='bg-inverse-surface text-inverse-muted hidden min-h-[340px] min-w-0 flex-col items-center justify-center gap-2.5 rounded-lg md:flex md:flex-[1_1_420px]'>
				<PinIcon />
				<p className='text-inverse-text font-semibold'>Яндекс Карта</p>
			</div>
		</div>
	)
}
