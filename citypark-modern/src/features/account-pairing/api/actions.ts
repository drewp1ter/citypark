'use server'

import { revalidatePath } from 'next/cache'
import { customerAuthorized, customerRequest, CustomerApiError, customerFallback, getCustomerSessionToken, getMiniSessionToken, type CustomerPairing } from '@/shared/customer'
import { pairingCodeSchema, type PairingState } from '../model/schema'

export async function redeemPairingAction(_previous: PairingState, form: FormData): Promise<PairingState> {
 const parsed = pairingCodeSchema.safeParse(String(form.get('code') ?? ''))
 if (!parsed.success) return { status: 'error', message: parsed.error.issues[0].message }
 try {
  const pairing = await customerAuthorized<CustomerPairing>('/customer/pairings/redeem', { method: 'POST', body: JSON.stringify({ code: parsed.data }) })
  if (!pairing) return { status: 'error', message: 'Войдите повторно' }
  return { status: 'pending', pairing }
 } catch (error) { return { status: 'error', message: error instanceof CustomerApiError ? error.message : customerFallback } }
}

export async function pairingStatusAction(id: string): Promise<PairingState> {
 if (!/^[A-Za-z0-9_-]{22}$/.test(id)) return { status: 'error', message: 'Код недействителен или устарел' }
 try {
  const pairing = await customerAuthorized<CustomerPairing>(`/customer/pairings/${id}`)
  if (!pairing) return { status: 'error', message: 'Войдите повторно' }
  return { status: pairing.status === 'completed' ? 'completed' : pairing.status === 'canceled' ? 'canceled' : 'pending', pairing }
 } catch (error) { return { status: 'error', message: error instanceof CustomerApiError ? error.message : customerFallback } }
}

export async function confirmPairingAction(id: string): Promise<PairingState> {
 if (!/^[A-Za-z0-9_-]{22}$/.test(id)) return { status: 'error', message: 'Код недействителен или устарел' }
 try {
  const pairing = await customerAuthorized<CustomerPairing>(`/customer/pairings/${id}/confirm`, { method: 'POST', body: '{}' })
  if (!pairing) return { status: 'error', message: 'Войдите повторно' }
  if (pairing.status === 'completed') { revalidatePath('/account'); revalidatePath('/account/orders') }
  return { status: pairing.status === 'completed' ? 'completed' : pairing.status === 'canceled' ? 'canceled' : 'pending', pairing }
 } catch (error) { return { status: 'error', message: error instanceof CustomerApiError ? error.message : customerFallback } }
}

export async function cancelPairingAction(id: string): Promise<PairingState> {
 if (!/^[A-Za-z0-9_-]{22}$/.test(id)) return { status: 'error', message: 'Код недействителен или устарел' }
 try {
  const result = await customerAuthorized<void>(`/customer/pairings/${id}`, { method: 'DELETE' })
  if (result === undefined && !(await getCustomerSessionToken())) return { status: 'error', message: 'Войдите повторно' }
  return { status: 'canceled' }
 } catch (error) { return { status: 'error', message: error instanceof CustomerApiError ? error.message : customerFallback } }
}

function miniError(error: unknown): PairingState {
 return { status: 'error', message: error instanceof CustomerApiError ? error.message : customerFallback }
}

async function miniRequest<T>(path: string, init: RequestInit = {}): Promise<T | undefined> {
 const token = await getMiniSessionToken()
 if (!token) return undefined
 return customerRequest<T>(path, { ...init, headers: { ...init.headers, Authorization: `Bearer ${token}` } })
}

export async function createMiniPairingAction(): Promise<PairingState> {
 try {
  const created = await miniRequest<{ id: string; code: string; expiresAt: string }>('/customer/pairings', { method: 'POST' })
  if (!created) return { status: 'error', message: 'Откройте приложение заново' }
  return { status: 'pending', code: created.code, pairing: { ...created, status: 'pending', sourceConfirmed: false, destinationConfirmed: false, otherName: '' } }
 } catch (error) { return miniError(error) }
}

export async function miniPairingStatusAction(id: string): Promise<PairingState> {
 if (!/^[A-Za-z0-9_-]{22}$/.test(id)) return { status: 'error', message: 'Код недействителен или устарел' }
 try {
  const pairing = await miniRequest<CustomerPairing>(`/customer/pairings/${id}`)
  if (!pairing) return { status: 'error', message: 'Откройте приложение заново' }
  return { status: pairing.status === 'completed' ? 'completed' : pairing.status === 'canceled' ? 'canceled' : 'pending', pairing }
 } catch (error) { return miniError(error) }
}

export async function confirmMiniPairingAction(id: string): Promise<PairingState> {
 if (!/^[A-Za-z0-9_-]{22}$/.test(id)) return { status: 'error', message: 'Код недействителен или устарел' }
 try {
  const pairing = await miniRequest<CustomerPairing>(`/customer/pairings/${id}/confirm`, { method: 'POST', body: '{}' })
  if (!pairing) return { status: 'error', message: 'Откройте приложение заново' }
  return { status: pairing.status === 'completed' ? 'completed' : pairing.status === 'canceled' ? 'canceled' : 'pending', pairing }
 } catch (error) { return miniError(error) }
}

export async function cancelMiniPairingAction(id: string): Promise<PairingState> {
 if (!/^[A-Za-z0-9_-]{22}$/.test(id)) return { status: 'error', message: 'Код недействителен или устарел' }
 try {
  const result = await miniRequest<void>(`/customer/pairings/${id}`, { method: 'DELETE' })
  if (result === undefined && !(await getMiniSessionToken())) return { status: 'error', message: 'Откройте приложение заново' }
  return { status: 'canceled' }
 } catch (error) { return miniError(error) }
}
