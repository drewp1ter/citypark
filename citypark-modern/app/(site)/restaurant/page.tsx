import type { Metadata } from 'next'

export const metadata: Metadata = { alternates: { canonical: '/restaurant' }, description: 'Ресторан и летнее кафе в Белореченске. Сити Парк, ул. Гоголя, 61.', title: 'Ресторан и летнее кафе — Сити Парк' }

export { RestaurantPage as default } from '@/pages/restaurant'
