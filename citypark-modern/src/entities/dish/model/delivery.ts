import { typo } from '@/shared/lib'
import type { DeliveryTown } from './types'
export type DeliveryInfo = { cooking: string; outskirtsFee: string; outskirtsRange: string; city: DeliveryTown | null }
export const defaultDeliveryInfo: DeliveryInfo = {
	cooking: typo('~30 мин'),
	outskirtsFee: typo('СОТ и ДНТ 200–350 ₽'),
	outskirtsRange: typo('200–350 ₽'),
	city: null,
}
