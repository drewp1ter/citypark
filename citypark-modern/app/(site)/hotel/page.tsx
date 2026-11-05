import type { Metadata } from 'next'

export const metadata: Metadata = { alternates: { canonical: '/hotel' }, description: 'Отель Сити Парк в Белореченске: информация об отеле и проживании.', title: 'Отель-люкс — Сити Парк' }

export { HotelPage as default } from '@/pages/hotel'
