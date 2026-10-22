import type { GuestDocument, GuestInfoSection, HotelRule } from '@/entities/hotel'
import { AccordionItem, cardClass } from '@/shared/ui'
import { emptySectionText, mobileCheckInRule } from '../model/content'

const item = 'border-b border-divider last:border-b-0'

/** Rules: six cards on desktop, an accordion on mobile. */
type RulesProps = {
	rules: readonly HotelRule[]
	/** On mobile the guest information and the documents are two more rows of the same accordion. */
	guestInfo: { intro: string; sections: readonly GuestInfoSection[]; documents: readonly GuestDocument[] }
}

export function Rules({ rules, guestInfo }: RulesProps) {
	const byTitle = (title: string) => rules.find((rule) => rule.title === title)?.text ?? ''

	return (
		<section className='max-w-page mx-auto w-full px-3 pt-11 md:px-6 md:pt-[104px]'>
			<h2 className='heading mb-6 text-5xl max-md:hidden'>Правила</h2>

			{/* Desktop */}
			<div className='grid gap-3 max-md:hidden md:grid-cols-2 lg:grid-cols-3'>
				{rules.map((rule, index) => (
					<div
						key={rule.title}
						className={cardClass(index === rules.length - 1 ? 'inverse' : 'surface', 'rounded-lg p-[26px]')}
					>
						<h3 className='mb-2 text-xl font-bold'>{rule.title}</h3>
						<p className={`text-[15px] ${index === rules.length - 1 ? 'text-inverse-muted' : 'text-text-muted'}`}>
							{rule.text}
						</p>
					</div>
				))}
			</div>

			{/* Mobile */}
			<h2 className='sr-only md:hidden'>Правила</h2>
			<div className='border-border bg-surface rounded-md border px-[18px] py-1 md:hidden'>
				<AccordionItem title='Заезд и выезд' defaultOpen className={item}>
					{mobileCheckInRule}
				</AccordionItem>
				<AccordionItem title='Оплата и бронь' className={item}>
					{byTitle('Оплата')} {byTitle('Гости')}
				</AccordionItem>
				<AccordionItem title='Дети, животные, курение' className={item}>
					{byTitle('Дети до 14 лет')} {byTitle('Нельзя')}
				</AccordionItem>
				<AccordionItem title='Информация для гостей' className={item}>
					<p className='mb-2'>{guestInfo.intro}</p>
					{guestInfo.sections.map((section) => (
						<AccordionItem key={section.title} title={section.title} className='border-divider border-t'>
							{section.text ?? emptySectionText}
						</AccordionItem>
					))}
				</AccordionItem>
				<AccordionItem title='Документы (PDF)' className={item}>
					<ul className='flex flex-col gap-2'>
						{guestInfo.documents.map((document) => (
							<li key={document.title}>
								<strong className='text-text'>{document.title}</strong> · {document.meta}
							</li>
						))}
					</ul>
				</AccordionItem>
			</div>
		</section>
	)
}
