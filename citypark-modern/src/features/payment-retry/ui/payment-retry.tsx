'use client'
import { useActionState, useEffect, useRef } from 'react'
import { Button } from '@/shared/ui'
import { retryPayment } from '../api/retry-payment'
import type { PaymentRetryProps, RetryState } from '../model/types'

export function PaymentRetry({ orderId, status, eligibility, surface }: PaymentRetryProps) {
 const [state, action, pending] = useActionState<RetryState, FormData>(retryPayment.bind(null, orderId, surface), {})
 const locked = useRef(false)
 useEffect(() => { if (!pending) locked.current = false }, [pending, state])
 const unpaid = status === 'new' || status === 'pending'
 const allowed = unpaid && eligibility?.allowed === true && eligibility.reason === ''
 const message = unpaid && eligibility?.allowed === false ? eligibility.reason === 'expired' ? 'Срок оплаты заказа истёк. Оформите новый заказ.' : eligibility.reason === 'closed' ? 'Оплата доступна только в часы работы ресторана.' : undefined : undefined
 return <div className='flex flex-col items-start gap-2'>
  {allowed && <form action={action} onSubmit={event => { if (pending || locked.current) event.preventDefault(); else locked.current = true }}>
   <Button type='submit' disabled={pending}>{pending ? 'Переходим к оплате…' : 'Оплатить'}</Button>
  </form>}
  <div role='status' aria-live='polite'>{state.error && <p>{state.error}</p>}{message && <p>{message}</p>}</div>
 </div>
}
