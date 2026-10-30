import type { CustomerPairing } from '@/shared/customer'

export function PairingStatus({ pairing, mini = false }: { pairing: CustomerPairing; mini?: boolean }) {
 return <div className='flex flex-col gap-2'>
  {pairing.otherName && <p>{mini ? 'Аккаунт сайта' : 'Аккаунт Telegram'}: {pairing.otherName}</p>}
  <p>В Telegram: {pairing.sourceConfirmed ? 'подтверждено' : 'ожидает подтверждения'} · На сайте: {pairing.destinationConfirmed ? 'подтверждено' : 'ожидает подтверждения'}</p>
  <p>Действует до {new Date(pairing.expiresAt).toLocaleString('ru-RU')}</p>
 </div>
}
