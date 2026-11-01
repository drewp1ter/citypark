'use client'
import { createContext, useContext, type ReactNode } from 'react'
import { defaultDeliveryInfo, type DeliveryInfo } from './delivery'
const Context = createContext<DeliveryInfo>(defaultDeliveryInfo)
export function DeliveryInfoProvider({ value, children }: { value: DeliveryInfo; children: ReactNode }) {
	return <Context value={value}>{children}</Context>
}
export function useDeliveryInfo() {
	return useContext(Context)
}
