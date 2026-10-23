import { typo } from '@/shared/lib'
import type { LegalDocument } from '../model/types'
import { LegalList } from './legal-list'

export function LegalPage({ document }: { document: LegalDocument }) {
	return (
		<main className='max-w-page mx-auto w-full px-3 pt-8 pb-16 md:px-6 md:pt-12'>
			<article className='mx-auto max-w-3xl'>
				<h1 className='font-display text-[32px] leading-none font-medium tracking-[-0.03em] md:text-[length:var(--text-h1)]'>
					{document.title}
				</h1>
				<div className='mt-6 leading-relaxed'>
					{document.intro.map(paragraph => (
						<p key={paragraph} className='mb-4'>
							{typo(paragraph)}
						</p>
					))}
					<LegalList items={document.items} root />
				</div>
			</article>
		</main>
	)
}
