import { Table } from '@heroui/react'
import { Link } from '@tanstack/react-router'
import type { components } from '../../../shared/api/schema'
import { formatDateTime, formatRub } from '../../../shared/lib/format'
import type { OrderSearch } from '../model/search'
import { PaymentChip, StatusChip } from './order-chips'

type Order = components['schemas']['Order']

const linkClass = 'font-medium text-accent hover:underline'

function Phone({ phone }: { phone: string }) {
	return (
		<a href={`tel:${phone}`} className='text-sm text-muted hover:underline'>
			{phone}
		</a>
	)
}

export function addressText(order: Order): string {
	return order.address === null ? 'Самовывоз' : `${order.address.city}, ${order.address.address}`
}

export function OrdersTable({ orders, search }: { orders: Order[]; search: OrderSearch }) {
	return (
		<>
			<Table className='hidden md:block'>
				<Table.ScrollContainer>
					<Table.Content aria-label='Заказы'>
						<Table.Header>
							<Table.Column isRowHeader>№</Table.Column>
							<Table.Column>Дата</Table.Column>
							<Table.Column>Клиент</Table.Column>
							<Table.Column>Адрес</Table.Column>
							<Table.Column>Сумма</Table.Column>
							<Table.Column>Оплата</Table.Column>
							<Table.Column>Статус</Table.Column>
						</Table.Header>
						<Table.Body>
							{orders.map((order) => (
								<Table.Row key={order.id} id={order.id}>
									<Table.Cell>
										<Link
											to='/orders/$orderId'
											params={{ orderId: String(order.id) }}
											search={search}
											className={linkClass}
											aria-label={`Заказ ${order.id}`}
										>
											{order.id}
										</Link>
									</Table.Cell>
									<Table.Cell>{formatDateTime(order.createdAt)}</Table.Cell>
									<Table.Cell>
										<div>{order.customer.name}</div>
										<Phone phone={order.customer.phone} />
									</Table.Cell>
									<Table.Cell>{addressText(order)}</Table.Cell>
									<Table.Cell>{formatRub(order.total + order.deliveryPrice)}</Table.Cell>
									<Table.Cell>
										<PaymentChip type={order.paymentType} />
									</Table.Cell>
									<Table.Cell>
										<StatusChip status={order.status} />
									</Table.Cell>
								</Table.Row>
							))}
						</Table.Body>
					</Table.Content>
				</Table.ScrollContainer>
			</Table>
			<ul className='flex flex-col gap-2 md:hidden'>
				{orders.map((order) => (
					<li key={order.id} className='relative rounded-xl border border-border bg-surface p-3'>
						<div className='flex items-center justify-between gap-2'>
							<Link
								to='/orders/$orderId'
								params={{ orderId: String(order.id) }}
								search={search}
								className={`${linkClass} after:absolute after:inset-0`}
								aria-label={`Заказ ${order.id}`}
							>
								№ {order.id}
							</Link>
							<StatusChip status={order.status} />
						</div>
						<div className='mt-1 text-sm text-muted'>{formatDateTime(order.createdAt)}</div>
						<div className='mt-2'>{order.customer.name}</div>
						<div className='mt-1 font-medium'>{formatRub(order.total + order.deliveryPrice)}</div>
					</li>
				))}
			</ul>
		</>
	)
}
