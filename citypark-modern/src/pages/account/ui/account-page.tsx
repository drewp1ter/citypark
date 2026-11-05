import { redirect } from 'next/navigation'
import { customerAuthorized, CustomerApiError, type CustomerMethod, type CustomerProfile } from '@/shared/customer'
import { Button } from '@/shared/ui'
import { logoutAction } from '../api/actions'
import { ProfileForm } from './profile-form'
import { MethodsPanel } from './methods-panel'
import { PairingPanel } from './pairing-panel'

export async function AccountPage({ searchParams }: { searchParams: Promise<{ linked?: string }> }) {
 let profile: CustomerProfile | undefined
 let methods: { methods: CustomerMethod[] } | undefined
 try {
  [profile, methods] = await Promise.all([
   customerAuthorized<CustomerProfile>('/customer/me'),
   customerAuthorized<{ methods: CustomerMethod[] }>('/customer/methods'),
  ])
 } catch (error) {
  if (error instanceof CustomerApiError && error.status === 401) redirect('/account/sign-in?returnTo=%2Faccount')
  return <main className='max-w-page mx-auto px-3 py-10'><h1 className='heading text-4xl'>Личный кабинет</h1><p role='alert'>Не удалось загрузить профиль. Попробуйте позже</p></main>
 }
 if (!profile || !methods) redirect('/account/sign-in?returnTo=%2Faccount')
 const linked = (await searchParams).linked
 return <main className='max-w-page mx-auto w-full px-3 py-10 md:px-6 md:py-16'><div className='mx-auto flex max-w-3xl flex-col gap-6'>
  <h1 className='heading text-4xl'>Личный кабинет</h1>
  {linked === 'google' && <p role='status'>Google подключён</p>}
  <ProfileForm profile={profile} />
  <MethodsPanel methods={methods.methods} email={profile.email} />
  {!methods.methods.some(method => method.provider === 'telegram') && <PairingPanel />}
  <div className='flex flex-wrap gap-3'><Button href='/account/addresses'>Мои адреса</Button><Button href='/account/orders'>История заказов</Button><form action={logoutAction}><Button type='submit' variant='outline'>Выйти</Button></form></div>
 </div></main>
}
