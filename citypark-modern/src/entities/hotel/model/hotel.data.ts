import { typo } from '@/shared/lib'
import type { GuestDocument } from './types'

// Copied from design/reference/*/hotel.html. The texts of the hotel and rooms pages are content blocks (entities/content).

/** Guest documents: scans on /documents and the legal texts. */
export const guestDocuments: readonly GuestDocument[] = [
	{ title: 'Выписка из реестра объектов классификации', meta: typo('2 стр.'), href: '/documents#classification' },
	{ title: 'Политика обработки персональных данных', meta: typo('8 стр.'), href: '/privacy' },
	{ title: 'Публичная оферта', meta: 'Текст', href: '/offer' },
]
