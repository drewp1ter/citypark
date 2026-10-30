'use client'

import { useActionState } from 'react'
import type { CustomerProfile } from '@/shared/customer'
import { Button, TextField, cardClass } from '@/shared/ui'
import { saveProfileAction } from '../api/actions'
import { initialAuthState } from '../model/schema'

export function ProfileForm({ profile }: { profile: CustomerProfile }) {
 const [state, action, pending] = useActionState(saveProfileAction, initialAuthState)
 return <form action={action} className={cardClass('surface', 'gap-5 rounded-lg p-5 md:p-8')}>
  <h2 className='font-display text-xl'>Профиль</h2>
  <TextField key={state.values?.name ?? profile.name} label='Имя' name='name' required maxLength={80} defaultValue={state.values?.name ?? profile.name} error={state.fields?.name} />
  <TextField key={state.values?.phone ?? profile.phone} label='Телефон (необязательно)' name='phone' type='tel' autoComplete='tel' defaultValue={state.values?.phone ?? profile.phone} error={state.fields?.phone} />
  {state.message && <p role={state.status === 'error' ? 'alert' : 'status'}>{state.message}</p>}
  <Button type='submit' disabled={pending}>Сохранить</Button>
 </form>
}
