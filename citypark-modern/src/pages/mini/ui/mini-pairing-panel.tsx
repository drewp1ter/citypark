'use client'

import { useEffect, useState, useTransition } from 'react'
import { AppLink, Button, cardClass } from '@/shared/ui'
import { createMiniPairingAction, miniPairingStatusAction, confirmMiniPairingAction, cancelMiniPairingAction, PairingStatus, type PairingState } from '@/features/account-pairing'

export function MiniPairingPanel() {
 const [state, setState] = useState<PairingState>({ status: 'idle' })
 const [busy, start] = useTransition()
 const pairing = state.pairing
 useEffect(() => {
  if (state.status !== 'pending' || !pairing) return
  let active = true
  let inFlight = false
  const timer = window.setInterval(async () => {
   if (!active || inFlight || document.visibilityState !== 'visible') return
   if (Date.now() >= new Date(pairing.expiresAt).getTime()) { setState({ status: 'error', message: 'Код недействителен или устарел' }); return }
   inFlight = true
   const next = await miniPairingStatusAction(pairing.id)
   inFlight = false
   if (active) setState(next.status === 'error' && next.message === 'Требуется авторизация' && pairing.sourceConfirmed ? { status: 'completed' } : next.status === 'error' ? { ...next, pairing, code: state.code } : { ...next, code: state.code })
  }, 3000)
  return () => { active = false; window.clearInterval(timer) }
 }, [state.status, state.code, pairing])
 const invoke = (action: (id: string) => Promise<PairingState>) => {
  if (!pairing || busy) return
  start(async () => {
   const next = await action(pairing.id)
   setState(next.status === 'error' ? { ...next, pairing, code: state.code } : { ...next, code: state.code })
  })
 }
 return <section className={cardClass('surface', 'gap-4 rounded-lg p-5')}>
  <h2 className='font-display text-xl'>Связать с сайтом</h2>
  <p>После подтверждения с обеих сторон способы входа и история заказов объединятся. Сеанс исходного аккаунта завершится.</p>
  {state.status === 'idle' && <Button type='button' disabled={busy} onClick={() => start(async () => setState(await createMiniPairingAction()))}>Создать код</Button>}
  {state.message && <p role='alert'>{state.message}</p>}
  {pairing && state.status === 'pending' && <div className='flex flex-col gap-3'>
   {state.code && <><p>Код для сайта: <strong className='break-all'>{state.code}</strong></p><Button type='button' variant='outline' onClick={() => void navigator.clipboard.writeText(state.code ?? '')}>Скопировать код</Button><p>Войдите в <AppLink href='/account' className='underline'>личный кабинет на сайте</AppLink> и вставьте код там.</p></>}
   <PairingStatus pairing={pairing} mini />
   {pairing.destinationConfirmed && <p>Сайт подтвердил связь. Подтвердите перенос аккаунта и истории заказов.</p>}
   <div className='flex flex-wrap gap-3'>
    {pairing.destinationConfirmed && !pairing.sourceConfirmed && <Button type='button' disabled={busy} onClick={() => invoke(confirmMiniPairingAction)}>Подтвердить связь</Button>}
    <Button type='button' variant='outline' disabled={busy} onClick={() => invoke(cancelMiniPairingAction)}>Отменить</Button>
   </div>
  </div>}
  {state.status === 'completed' && <p role='status'>Связь подтверждена. Откройте приложение заново</p>}
  {state.status === 'canceled' && <p role='status'>Связь отменена.</p>}
  {(state.status === 'error' || state.status === 'canceled') && <Button type='button' variant='outline' onClick={() => setState({ status: 'idle' })}>Создать новый код</Button>}
  {state.status === 'error' && <p>Если вход устарел, закройте и снова откройте приложение через Telegram.</p>}
 </section>
}
