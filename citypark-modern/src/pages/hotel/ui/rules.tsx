import type { GuestDocument } from '@/entities/hotel'
import { AccordionItem, cardClass, RichText } from '@/shared/ui'
import { emptySectionText } from '../model/content'
import type { GuestInfoCopy, TitledHtml } from '../model/types'

const item = 'border-b border-divider last:border-b-0'

/** Rules: six cards on desktop, an accordion on mobile. */
type RulesProps = {
	heading: string
	rules: readonly TitledHtml[]
	/** The mobile wording of the check-in rule, placeholders already filled. */
	mobileCheckInRule: string
	/** On mobile the guest information and the documents are two more rows of the same accordion. */
	guestInfo: GuestInfoCopy
	documents: readonly GuestDocument[]
}

// The mobile accordion joins the rules by their position in the list (the desktop order of the blocks).
const [PAYMENT, GUESTS, CHILDREN, FORBIDDEN] = [2, 3, 4, 5]

export function Rules({ heading, rules, mobileCheckInRule, guestInfo, documents }: RulesProps) {
	/** Rule texts side by side, as one paragraph. */
	const joined = (...indexes: number[]) => (
		<RichText
			html={indexes.map((index) => rules[index]?.html ?? '').join(' ')}
			variant='short'
			className='[&_p]:inline'
		/>
	)

	return (
		<section className='max-w-page mx-auto w-full px-3 pt-11 md:px-6 md:pt-[104px]'>
			<h2 className='heading mb-6 text-5xl max-md:hidden'>{heading}</h2>

			{/* Desktop */}
			<div className='grid gap-3 max-md:hidden md:grid-cols-2 lg:grid-cols-3'>
				{rules.map((rule, index) => (
					<div
						key={rule.title}
						className={cardClass(index === rules.length - 1 ? 'inverse' : 'surface', 'rounded-lg p-[26px]')}
					>
						<h3 className='mb-2 text-xl font-bold'>{rule.title}</h3>
						<RichText
							html={rule.html}
							variant='short'
							className={`text-[15px] ${index === rules.length - 1 ? 'text-inverse-muted' : 'text-text-muted'}`}
						/>
					</div>
				))}
			</div>

			{/* Mobile */}
			<h2 className='sr-only md:hidden'>{heading}</h2>
			<div className='border-border bg-surface rounded-md border px-[18px] py-1 md:hidden'>
				<AccordionItem title='Заезд и выезд' defaultOpen className={item}>
					{mobileCheckInRule}
				</AccordionItem>
				<AccordionItem title='Оплата и бронь' className={item}>
					{joined(PAYMENT, GUESTS)}
				</AccordionItem>
				<AccordionItem title='Дети, животные, курение' className={item}>
					{joined(CHILDREN, FORBIDDEN)}
				</AccordionItem>
				<AccordionItem title='Информация для гостей' className={item}>
					<p className='mb-2'>{guestInfo.intro}</p>
					{guestInfo.sections.map((section) => (
						<AccordionItem key={section.title} title={section.title} className='border-divider border-t'>
							{section.html ? <RichText html={section.html} variant='short' /> : emptySectionText}
						</AccordionItem>
					))}
				</AccordionItem>
				<AccordionItem title='Документы (PDF)' className={item}>
					<ul className='flex flex-col gap-2'>
						{documents.map((document) => (
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
