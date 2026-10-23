import { typo } from '@/shared/lib'
import type { GuestDocument } from './types'

// Copied from design/reference/*/hotel.html. The texts of the hotel and rooms pages are content blocks (entities/content).

/** Guest documents. They are files, not blocks; the PDFs have not been sent yet. */
export const guestDocuments: readonly GuestDocument[] = [
	{ title: 'Выписка из реестра объектов классификации', meta: typo('PDF · 2 стр.') },
	{ title: 'Политика обработки персональных данных', meta: typo('PDF · 8 стр.') },
	{ title: 'Публичная оферта', meta: 'PDF' },
]
