import { typo } from '@/shared/lib'
import type { LegalItem } from '../model/types'

// Numbers 1. / 1.1. / 1.1.1. come from CSS counters, so the markup stays a real <ol> for screen readers.
export function LegalList({ items, root = false }: { items: readonly LegalItem[]; root?: boolean }) {
	return (
		<ol className={`legal-list${root ? '' : ' mt-2'} space-y-2`}>
			{items.map(item => (
				<li key={item.text} className={root ? 'mt-4 font-semibold' : 'font-normal'}>
					{typo(item.text)}
					{item.note && <p className='mt-2 font-normal'>{typo(item.note)}</p>}
					{item.items && <LegalList items={item.items} />}
				</li>
			))}
		</ol>
	)
}
