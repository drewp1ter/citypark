'use client'

import { useActionState, useEffect, useState, useTransition } from 'react'
import { useRouter } from 'next/navigation'
import { AppLink, Button, TextField, cardClass } from '@/shared/ui'
import { cancelPairingAction, confirmPairingAction, pairingStatusAction, redeemPairingAction, PairingStatus, type PairingState } from '@/features/account-pairing'

const initial: PairingState = { status: 'idle' }

export function PairingPanel() {
 const [redeemed, redeem, redeemPending] = useActionState(redeemPairingAction, initial)
 const [current, setCurrent] = useState<PairingState | null>(null)
 const [busy, startTransition] = useTransition()
 const router = useRouter()
 const state = current ?? redeemed
 const pairing = state.pairing
 useEffect(() => {
  if (state.status !== 'pending' || !pairing) return
  let active = true
  let inFlight = false
  const expires = new Date(pairing.expiresAt).getTime()
  const timer = window.setInterval(async () => {
   if (!active || inFlight || document.visibilityState !== 'visible') return
   if (Date.now() >= expires) { setCurrent({ status: 'error', message: 'Код недействителен или устарел' }); return }
   inFlight = true
   const next = await pairingStatusAction(pairing.id)
   inFlight = false
   if (active) { setCurrent(next.status === 'error' ? { ...next, pairing } : next); if (next.status === 'completed') router.refresh() }
  }, 3000)
  return () => { active = false; window.clearInterval(timer) }
 }, [state.status, pairing, router])
 const invoke = (action: (id: string) => Promise<PairingState>) => {
  if (!pairing || busy) return
  startTransition(async () => {
   const next = await action(pairing.id)
   setCurrent(next.status === 'error' ? { ...next, pairing } : next)
   if (next.status === 'completed') router.refresh()
  })
 }
 return <section className={cardClass('surface', 'gap-5 rounded-lg p-5 md:p-8')}>
  <h2 className='font-display text-xl'>Подключить Telegram</h2>
  <p>Создайте код в Telegram Mini App и введите его здесь. После подтверждения с обеих сторон заказы и способы входа объединятся. Сеансы исходного Telegram аккаунта завершатся.</p>
  {!pairing && state.status !== 'completed' && <form action={redeem} onSubmit={() => setCurrent(null)} className='flex flex-col gap-3'>
   <TextField label='Код из Telegram' name='code' required minLength={22} maxLength={22} autoComplete='off' />
   <Button type='submit' disabled={redeemPending}>Продолжить</Button>
  </form>}
  {state.message && <p role='alert'>{state.message}</p>}
  {pairing && state.status === 'pending' && <div className='flex flex-col gap-3'>
   <PairingStatus pairing={pairing} />
   <div className='flex flex-wrap gap-3'>
    {!pairing.destinationConfirmed && <Button type='button' disabled={busy} onClick={() => invoke(confirmPairingAction)}>Подтвердить связь</Button>}
    <Button type='button' variant='outline' disabled={busy} onClick={() => invoke(cancelPairingAction)}>Отменить</Button>
   </div>
  </div>}
  {state.status === 'completed' && <p role='status'>Аккаунты связаны. История заказов обновлена.</p>}
  {state.status === 'canceled' && <p role='status'>Связь отменена.</p>}
  {(state.status === 'canceled' || state.status === 'error') && <Button type='button' variant='outline' onClick={() => setCurrent(initial)}>Ввести новый код</Button>}
  {state.status === 'error' && pairing && <Button type='button' variant='outline' onClick={() => invoke(pairingStatusAction)}>Повторить проверку</Button>}
  {state.status === 'error' && <AppLink href='/account/sign-in?returnTo=%2Faccount' className='min-h-11 self-start py-2 underline'>Войти повторно</AppLink>}
 </section>
}
