'use client'

import { useActionState } from 'react'
import type { CustomerMethod } from '@/shared/customer'
import { AppLink, Button, TextField, cardClass } from '@/shared/ui'
import { changeEmailAction } from '../api/actions'
import { startGoogleLink } from '../api/google-actions'
import { initialAuthState } from '../model/schema'

const labels = { email: 'Email и пароль', google: 'Google', telegram: 'Telegram' }

export function MethodsPanel({ methods, email }: { methods: CustomerMethod[]; email: string }) {
 const [state, action, pending] = useActionState(changeEmailAction, initialAuthState)
 const hasEmail = methods.some(method => method.provider === 'email')
 const hasGoogle = methods.some(method => method.provider === 'google')
 return <section className={cardClass('surface', 'gap-5 rounded-lg p-5 md:p-8')}>
  <h2 className='font-display text-xl'>Способы входа</h2>
  {methods.length ? <ul className='flex flex-col gap-2'>{methods.map(method => <li key={method.provider}>{labels[method.provider]} — {method.verified ? 'подключён' : 'не подтверждён'}</li>)}</ul> : <p>Способы входа не найдены</p>}
  {hasEmail && <form action={action} className='flex flex-col gap-3'>
   <p>Текущий email: {email}</p>
   <TextField label='Новый email' name='email' type='email' autoComplete='email' required error={state.fields?.email} defaultValue={state.values?.email} />
   {state.message && <p role={state.status === 'error' ? 'alert' : 'status'}>{state.message}</p>}
   <Button type='submit' variant='outline' disabled={pending}>Отправить подтверждение</Button>
  </form>}
  {!hasGoogle && <form action={startGoogleLink}><Button type='submit' variant='outline'>Подключить Google</Button></form>}
  <AppLink href='/account/sign-in' className='min-h-11 self-start py-2 underline'>Войти повторно для изменения способов входа</AppLink>
 </section>
}
