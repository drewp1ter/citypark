import { Button, Skeleton, toast } from '@heroui/react'
import { useNavigate } from '@tanstack/react-router'
import { useCallback, useEffect, useRef, useState } from 'react'
import { PageHeader } from '../../../shared/ui/page-header'
import { useLiveOrders } from '../api'
import { alertText, isPaidAlert, titleWithCount } from '../model/alerts'
import type { LiveOrderEvent } from '../model/socket'
import { ConnectionStatus } from './connection-status'
import { OrderColumn } from './order-column'

const TITLE = 'Заказы онлайн'
const FRESH_MS = 10_000

function canNotify(): boolean {
	return typeof window !== 'undefined' && 'Notification' in window
}

export function LiveOrdersPage() {
	const navigate = useNavigate()
	const [freshIds, setFreshIds] = useState<ReadonlySet<number>>(new Set())
	const [missed, setMissed] = useState(0)
	const [permission, setPermission] = useState(() => (canNotify() ? Notification.permission : 'denied'))
	const timers = useRef(new Set<ReturnType<typeof setTimeout>>())

	const openOrder = useCallback(
		(id: number) => void navigate({ to: '/orders/$orderId', params: { orderId: String(id) } }),
		[navigate]
	)

	const onOrderEvent = useCallback(
		(e: LiveOrderEvent) => {
			const order = e.order
			if (order === undefined) return
			setFreshIds((ids) => new Set(ids).add(order.id))
			const timer = setTimeout(() => {
				timers.current.delete(timer)
				setFreshIds((ids) => {
					const next = new Set(ids)
					next.delete(order.id)
					return next
				})
			}, FRESH_MS)
			timers.current.add(timer)

			if (!isPaidAlert(e)) return
			toast.success(alertText(order), {
				actionProps: { children: 'Открыть', onPress: () => openOrder(order.id) },
			})
			if (!document.hidden) return
			setMissed((n) => n + 1)
			if (canNotify() && Notification.permission === 'granted') {
				const notification = new Notification(alertText(order))
				notification.onclick = () => {
					window.focus()
					openOrder(order.id)
					notification.close()
				}
			}
		},
		[openOrder]
	)

	const { board, status, isLoading, error, refetch } = useLiveOrders(onOrderEvent)

	useEffect(() => {
		const pending = timers.current
		return () => pending.forEach(clearTimeout)
	}, [])

	useEffect(() => {
		const reset = () => {
			if (!document.hidden) setMissed(0)
		}
		document.addEventListener('visibilitychange', reset)
		return () => document.removeEventListener('visibilitychange', reset)
	}, [])

	useEffect(() => {
		const previous = document.title
		document.title = titleWithCount(TITLE, missed)
		return () => {
			document.title = previous
		}
	}, [missed])

	async function enableNotifications() {
		setPermission(await Notification.requestPermission())
	}

	let content
	if (isLoading) {
		content = (
			<div className='grid gap-4 lg:grid-cols-2' aria-busy='true' aria-label='Загрузка'>
				{[0, 1].map((n) => (
					<Skeleton key={n} className='h-64 w-full rounded-2xl' />
				))}
			</div>
		)
	} else if (error !== null) {
		content = (
			<div role='alert' className='flex flex-col items-start gap-3'>
				<p className='text-danger'>{error instanceof Error ? error.message : 'Не удалось загрузить заказы'}</p>
				<Button variant='secondary' onPress={refetch}>
					Повторить
				</Button>
			</div>
		)
	} else {
		content = (
			<div className='grid items-start gap-4 lg:grid-cols-2'>
				<OrderColumn title='Ждут оплаты' orders={board.pending} freshIds={freshIds} />
				<OrderColumn title='Новые' orders={board.fresh} freshIds={freshIds} />
			</div>
		)
	}

	return (
		<>
			<PageHeader
				title={TITLE}
				actions={
					<>
						<ConnectionStatus status={status} />
						{canNotify() && permission === 'default' && (
							<Button variant='secondary' onPress={() => void enableNotifications()}>
								Включить уведомления
							</Button>
						)}
					</>
				}
			/>
			{content}
		</>
	)
}
