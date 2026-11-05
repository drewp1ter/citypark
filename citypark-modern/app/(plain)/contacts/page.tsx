import type { Metadata } from 'next'

export const metadata: Metadata = { alternates: { canonical: '/contacts' }, description: 'Адрес и телефоны отеля, ресторана и доставки Сити Парк в Белореченске.', title: 'Контакты — Сити Парк' }

export { ContactsPage as default } from '@/pages/contacts'
