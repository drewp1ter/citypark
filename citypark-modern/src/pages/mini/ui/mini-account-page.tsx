import { getMiniProfile } from '@/features/telegram-launch'
import { AppLink, Button } from '@/shared/ui'
import { MiniPairingPanel } from './mini-pairing-panel'

export async function MiniAccountPage() {
 const profile = await getMiniProfile()
 if (!profile) return null
 return <main className='mx-auto flex max-w-3xl flex-col gap-6 px-4 py-6 pb-[var(--mini-bottom,env(safe-area-inset-bottom))]'>
  <h1 className='heading text-3xl'>Аккаунт Telegram</h1>
  <p>Вы вошли как {profile.name}</p>
  <Button href='/mini/orders' className='self-start'>История заказов</Button>
  <Button href='/mini/addresses' className='self-start'>Мои адреса</Button>
  <MiniPairingPanel />
  <p>Google можно подключить в <AppLink href='/account' className='underline'>личном кабинете на сайте</AppLink>.</p>
  <AppLink href='/mini' className='min-h-11 py-2 underline'>В меню</AppLink>
 </main>
}
