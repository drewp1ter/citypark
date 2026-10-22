import type { GuestDocument, GuestInfoSection } from '@/entities/hotel'
import { AccordionItem, AppLink, cardClass } from '@/shared/ui'
import { emptySectionText } from '../model/content'

type GuestInfoProps = {
	intro: string
	sections: readonly GuestInfoSection[]
	documents: readonly GuestDocument[]
}

function DocumentCard({ document }: { document: GuestDocument }) {
	const content = (
		<>
			<span className='rounded-pill bg-bg self-start px-2.5 py-0.5 text-xs font-semibold'>{document.meta}</span>
			<strong>{document.title}</strong>
		</>
	)
	const classes = cardClass('surface', 'gap-1.5 rounded-md p-5')

	// The PDF files have not been sent yet; until then a document is a plain card, not a link.
	return document.href ? (
		<AppLink href={document.href} className={classes}>
			{content}
		</AppLink>
	) : (
		<div className={classes}>{content}</div>
	)
}

/** "Информация для гостей" (accordion) and the document list (PDF). */
export function GuestInfo({ intro, sections, documents }: GuestInfoProps) {
	return (
		<section
			id='info'
			className='max-w-page mx-auto grid w-full items-start gap-3 px-3 pt-11 pb-6 max-md:hidden md:px-6 md:pt-[104px] lg:grid-cols-3'
		>
			<div className={cardClass('surface', 'rounded-lg p-5 md:rounded-xl md:p-8 lg:col-span-2')}>
				<h2 className='font-display mb-1 text-[22px] font-medium md:text-[30px]'>Информация для гостей</h2>
				<p className='text-text-muted mb-4 text-sm'>{intro}</p>
				<div>
					{sections.map((section, index) => (
						<AccordionItem
							key={section.title}
							title={section.title}
							defaultOpen={index === 0}
							className='border-divider border-t'
						>
							{section.text ?? emptySectionText}
						</AccordionItem>
					))}
				</div>
			</div>

			<div id='docs' className='flex flex-col gap-3'>
				<h2 className='sr-only'>Документы</h2>
				{documents.map((document) => (
					<DocumentCard key={document.title} document={document} />
				))}
			</div>
		</section>
	)
}
