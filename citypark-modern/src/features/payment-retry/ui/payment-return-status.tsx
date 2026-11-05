'use client'
import { useRouter } from 'next/navigation'
import { Button } from '@/shared/ui'

export function PaymentReturnStatus({ status, payment }: { status: string; payment?: string }) {
 const router = useRouter()
 if (status === 'paid' || status === 'confirmed') return <p role='status'>Заказ оплачен</p>
 if (status !== 'new' && status !== 'pending') return null
 if (payment === 'failed') return <p role='status'>Оплата не завершена</p>
 if (payment !== 'returned') return null
 return <div className='flex flex-col items-start gap-3'><p role='status'>Платёж ещё обрабатывается</p><Button variant='outline' onClick={() => router.refresh()}>Обновить статус</Button></div>
}
