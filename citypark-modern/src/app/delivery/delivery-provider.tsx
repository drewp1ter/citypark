import type { ReactNode } from 'react'
import { DeliveryInfoProvider, getDeliveryInfo } from '@/entities/dish'
export async function DeliveryProvider({ children }: { children: ReactNode }) {
	return <DeliveryInfoProvider value={await getDeliveryInfo()}>{children}</DeliveryInfoProvider>
}
