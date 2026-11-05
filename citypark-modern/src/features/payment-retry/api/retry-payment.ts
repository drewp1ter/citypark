'use server'
import { redirect } from 'next/navigation'
import { revalidatePath } from 'next/cache'
import { customerRequest, CustomerApiError, getCustomerSessionToken, getMiniSessionToken, setPendingPaymentContext } from '@/shared/customer'
import type { RetryState } from '../model/types'

const fallback = 'Не удалось перейти к оплате. Попробуйте ещё раз.'
const closed = 'Оплата доступна только в часы работы ресторана.'

export async function retryPayment(orderId: number, surface: 'web' | 'mini', _previous: RetryState, _form: FormData): Promise<RetryState> {
 void _previous
 void _form
 if (!Number.isSafeInteger(orderId) || orderId < 1 || (surface !== 'web' && surface !== 'mini')) return { error: fallback }
 const path = `/${surface === 'mini' ? 'mini' : 'account'}/orders/${orderId}`
 const login = surface === 'mini' ? '/mini' : `/account/sign-in?returnTo=${encodeURIComponent(path)}`
 const token = surface === 'mini' ? await getMiniSessionToken() : await getCustomerSessionToken()
 if (!token) redirect(login)
 let target: string | undefined
 let expired = false
 try {
  const result = await customerRequest<{ orderId: number; redirectUrl: string }>(`/orders/${orderId}/payment-retry`, { method: 'POST', headers: { Authorization: `Bearer ${token}` } })
  if (result.orderId !== orderId || typeof result.redirectUrl !== 'string' || !result.redirectUrl.trim()) return { error: fallback }
  const url = new URL(result.redirectUrl)
  if (!['https:', 'http:'].includes(url.protocol) || url.username || url.password) return { error: fallback }
  await setPendingPaymentContext(orderId, surface, 'history')
  target = result.redirectUrl
 } catch (error) {
  if (error instanceof CustomerApiError) {
   if (error.status === 401) expired = true
   else if (error.status === 404) return { error: 'Заказ недоступен.' }
   else if (error.status === 409 || (error.status === 503 && error.message === closed)) {
    revalidatePath(path)
    revalidatePath(`/${surface === 'mini' ? 'mini' : 'account'}/orders`)
    return { error: error.message }
   }
  }
  if (!expired) return { error: fallback }
 }
 if (expired) redirect(login)
 if (target) redirect(target)
 return { error: fallback }
}
