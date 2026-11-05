import type { Metadata } from 'next'

export const metadata: Metadata = { alternates: { canonical: '/news' }, description: 'Новости отеля и ресторана Сити Парк в Белореченске.', title: 'Новости — Сити Парк' }

export { NewsPage as default } from '@/pages/news'
