import { Button, Drawer, Skeleton, Table } from '@heroui/react'
import { useQuery } from '@tanstack/react-query'
import { ApiError } from '../../../shared/api'
import type { components } from '../../../shared/api/schema'
import { formatDateTime, formatRub } from '../../../shared/lib/format'
import { orderQuery } from '../api'
import { PaymentChip, StatusChip } from './order-chips'
import { addressText } from './orders-table'

type Order = components['schemas']['Order']

function Field({ label, children }: { label: string; children: React.ReactNode }) {
	return (
		<div>
			<dt className='text-sm text-muted'>{label}</dt>
			<dd>{children}</dd>
		</div>
	)
}

function OrderBody({ order }: { order: Order }) {
	return (
		<div className='flex flex-col gap-4'>
			<div className='flex flex-wrap items-center gap-2'>
				<StatusChip status={order.status} />
				<PaymentChip type={order.paymentType} />
				<span className='text-sm text-muted'>{formatDateTime(order.createdAt)}</span>
			</div>
			<dl className='grid gap-3 sm:grid-cols-2'>
				<Field label='Клиент'>{order.customer.name}</Field>
				<Field label='Телефон'>
					<a href={`tel:${order.customer.phone}`} className='text-accent hover:underline'>
						{order.customer.phone}
					</a>
				</Field>
				<Field label='Адрес'>{addressText(order)}</Field>
				{order.comment !== '' && <Field label='Комментарий'>{order.comment}</Field>}
			</dl>
			<Table>
				<Table.ScrollContainer>
					<Table.Content aria-label='Состав заказа'>
						<Table.Header>
							<Table.Column isRowHeader>Блюдо</Table.Column>
							<Table.Column>Категория</Table.Column>
							<Table.Column>Кол-во</Table.Column>
							<Table.Column>Цена</Table.Column>
							<Table.Column>Сумма</Table.Column>
						</Table.Header>
						<Table.Body>
							{order.items.map((item) => (
								<Table.Row key={item.productId} id={item.productId}>
									<Table.Cell>{item.name}</Table.Cell>
									<Table.Cell>{item.categoryName}</Table.Cell>
									<Table.Cell>{item.amount}</Table.Cell>
									<Table.Cell>{formatRub(item.offeredPrice)}</Table.Cell>
									<Table.Cell>{formatRub(item.offeredPrice * item.amount)}</Table.Cell>
								</Table.Row>
							))}
						</Table.Body>
					</Table.Content>
				</Table.ScrollContainer>
			</Table>
			<dl className='flex flex-col gap-1 text-right'>
				<div>
					<dt className='inline text-muted'>Доставка: </dt>
					<dd className='inline'>{formatRub(order.deliveryPrice)}</dd>
				</div>
				<div className='text-lg font-semibold'>
					<dt className='inline'>Итого: </dt>
					<dd className='inline'>{formatRub(order.total + order.deliveryPrice)}</dd>
				</div>
			</dl>
		</div>
	)
}

/** The order drawer: a side panel on desktop, full width on phones. */
export function OrderDetails({ orderId, onClose }: { orderId: string; onClose: () => void }) {
	const id = /^\d+$/.test(orderId) ? Number(orderId) : null
	const query = useQuery({ ...orderQuery(id ?? 0), enabled: id !== null })
	const notFound = id === null || (query.error instanceof ApiError && query.error.status === 404)

	let body
	if (notFound) {
		body = <p>Заказ не найден</p>
	} else if (query.isError) {
		body = (
			<div role='alert' className='flex flex-col items-start gap-3'>
				<p className='text-danger'>{query.error.message}</p>
				<Button variant='secondary' onPress={() => void query.refetch()}>
					Повторить
				</Button>
			</div>
		)
	} else if (query.isPending) {
		body = (
			<div className='flex flex-col gap-2' aria-busy='true' aria-label='Загрузка'>
				{[0, 1, 2].map((n) => (
					<Skeleton key={n} className='h-10 w-full rounded-lg' />
				))}
			</div>
		)
	} else {
		body = <OrderBody order={query.data} />
	}

	return (
		<Drawer.Backdrop isOpen onOpenChange={(open) => !open && onClose()}>
			<Drawer.Content placement='right'>
				<Drawer.Dialog aria-label='Заказ' className='w-full sm:max-w-2xl'>
					<Drawer.CloseTrigger />
					<Drawer.Header>
						<Drawer.Heading>{notFound ? 'Заказ' : `Заказ № ${orderId}`}</Drawer.Heading>
					</Drawer.Header>
					<Drawer.Body>{body}</Drawer.Body>
					<Drawer.Footer>
						<Button variant='secondary' onPress={onClose}>
							Назад к списку
						</Button>
					</Drawer.Footer>
				</Drawer.Dialog>
			</Drawer.Content>
		</Drawer.Backdrop>
	)
}
