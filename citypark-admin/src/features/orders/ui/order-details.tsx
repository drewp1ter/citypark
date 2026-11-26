import { Button, Drawer, Skeleton, Table, TextArea } from '@heroui/react'
import { useQuery } from '@tanstack/react-query'
import { ApiError } from '../../../shared/api'
import type { components } from '../../../shared/api/schema'
import { formatDateTime, formatRub } from '../../../shared/lib/format'
import { useState } from 'react'
import { toastError } from '../../../shared/ui/toast'
import { iikoStatusLabel } from '../../../shared/lib/iiko-status'
import { orderQuery, useApproveOrder } from '../api'
import { StatusChip } from '../../../shared/ui/order-chips'
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
	const hasPromo = order.promoCode !== null
	const editable = order.iikoStatus === 'awaiting_approval'
	const [comments, setComments] = useState<Record<number, string>>({})
	const [excluded, setExcluded] = useState<number[]>([])
	const approval = useApproveOrder(order.id)
	const included = order.items.filter((item) =>
		editable ? !excluded.includes(item.productId) : !item.excludedFromIiko
	)
	const foodTotal =
		included.reduce(
			(total, item) => total + Math.round(item.offeredPrice * 100) * item.amount - Math.round(item.discount * 100),
			0
		) / 100
	const itemComments = included.flatMap(({ productId }) => {
		const comment = comments[productId]?.trim()
		return comment ? [{ productId, comment }] : []
	})
	const tooLong = itemComments.some(({ comment }) => Array.from(comment).length > 255)
	const errors = approval.error instanceof ApiError ? approval.error.fields : {}
	return (
		<div className='flex flex-col gap-4'>
			<div className='flex flex-wrap items-center gap-2'>
				<StatusChip status={order.status} />
				{order.iikoStatus && (
					<span role='status' className='text-sm'>
						{iikoStatusLabel(order.iikoStatus)}
					</span>
				)}
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
							<Table.Column>{hasPromo ? 'До скидки' : 'Сумма'}</Table.Column>
							{hasPromo && <Table.Column>Скидка по промокоду</Table.Column>}
						</Table.Header>
						<Table.Body>
							{order.items.map((item) => {
								const id = item.productId
								const omitted = editable ? excluded.includes(id) : item.excludedFromIiko
								const comment = comments[id] ?? ''
								const commentError =
									Array.from(comment.trim()).length > 255 ? 'Введите до 255 символов.' : errors[`itemComments.${id}`]
								return (
									<Table.Row key={item.productId} id={item.productId}>
										<Table.Cell>
											<div className='flex min-w-40 flex-col items-start gap-1'>
												<span>{item.name}</span>
												{item.isOnStopList && <span className='text-sm text-warning'>В стоп-листе iiko</span>}
												{omitted && (
													<span className='text-sm text-muted'>
														{editable ? 'Не будет отправлено в iiko' : 'Исключено из отправки в iiko'}
													</span>
												)}
												{editable && (omitted || item.isOnStopList) && (
													<Button
														size='sm'
														variant='secondary'
														aria-label={`${omitted ? 'Включить в отправку' : 'Не создавать в iiko'} — ${item.name}`}
														isDisabled={approval.isPending}
														onPress={() =>
															setExcluded((ids) => (omitted ? ids.filter((value) => value !== id) : [...ids, id]))
														}
													>
														{omitted ? 'Включить в отправку' : 'Не создавать в iiko'}
													</Button>
												)}
												{editable && !omitted && (
													<>
														<label htmlFor={`iiko-comment-${id}`} className='text-sm'>
															Комментарий для кухни
														</label>
														<TextArea
															id={`iiko-comment-${id}`}
															value={comment}
															aria-label={`Комментарий для кухни — ${item.name}`}
															aria-describedby={commentError ? `iiko-comment-error-${id}` : undefined}
															aria-invalid={Boolean(commentError)}
															disabled={approval.isPending}
															onChange={(event) => setComments((draft) => ({ ...draft, [id]: event.target.value }))}
														/>
														{commentError && (
															<span id={`iiko-comment-error-${id}`} className='text-sm text-danger'>
																{commentError}
															</span>
														)}
													</>
												)}
												{!editable && item.iikoComment && (
													<span className='text-sm wrap-anywhere whitespace-pre-wrap'>{item.iikoComment}</span>
												)}
											</div>
										</Table.Cell>
										<Table.Cell>{item.categoryName}</Table.Cell>
										<Table.Cell>{item.amount}</Table.Cell>
										<Table.Cell>{formatRub(item.offeredPrice)}</Table.Cell>
										<Table.Cell>{formatRub(item.offeredPrice * item.amount)}</Table.Cell>
										{hasPromo && <Table.Cell>{formatRub(0 - item.discount)}</Table.Cell>}
									</Table.Row>
								)
							})}
						</Table.Body>
					</Table.Content>
				</Table.ScrollContainer>
			</Table>
			<dl aria-label='Итоги заказа' className='flex flex-col gap-1 text-right'>
				{hasPromo && (
					<>
						<div>
							<dt className='inline text-muted'>Блюда до скидки: </dt>
							<dd className='inline'>{formatRub(order.subtotal)}</dd>
						</div>
						<div>
							<dt className='inline text-muted'>Промокод: </dt>
							<dd className='inline wrap-anywhere'>{order.promoCode}</dd>
						</div>
						<div>
							<dt className='inline text-muted'>Скидка по промокоду: </dt>
							<dd className='inline'>{formatRub(0 - order.discount)}</dd>
						</div>
						<div>
							<dt className='inline text-muted'>Блюда со скидкой: </dt>
							<dd className='inline'>{formatRub(order.total)}</dd>
						</div>
					</>
				)}
				<div>
					<dt className='inline text-muted'>Доставка: </dt>
					<dd className='inline'>{formatRub(order.deliveryPrice)}</dd>
				</div>
				<div className='text-lg font-semibold'>
					<dt className='inline'>{order.iikoStatus ? 'Оплачено: ' : 'Итого: '}</dt>
					<dd className='inline'>{formatRub(order.total + order.deliveryPrice)}</dd>
				</div>
				{order.iikoStatus && (
					<div>
						<dt className='inline text-muted'>Блюда для iiko: </dt>
						<dd className='inline'>{formatRub(foodTotal)}</dd>
					</div>
				)}
			</dl>
			{editable && (
				<div className='flex flex-col items-start gap-2'>
					{included.length === 0 && <p>Для отправки выберите хотя бы одно блюдо</p>}
					<Button
						isPending={approval.isPending}
						isDisabled={approval.isPending || included.length === 0 || tooLong}
						onPress={() => approval.mutate({ excludedProductIds: excluded, itemComments }, { onError: toastError })}
					>
						Отправить в IIKO
					</Button>
				</div>
			)}
		</div>
	)
}

/** The order drawer: a side panel on desktop, full width on phones. */
export function OrderDetails({
	orderId,
	onClose,
	closeLabel = 'Назад к списку',
}: {
	orderId: string
	onClose: () => void
	closeLabel?: string
}) {
	const id = /^\d+$/.test(orderId) ? Number(orderId) : null
	const query = useQuery({ ...orderQuery(id ?? 0), enabled: id !== null })
	const notFound = id === null || (query.error instanceof ApiError && query.error.status === 404)

	let body
	if (notFound) {
		body = <p>Заказ не найден</p>
	} else if (query.isLoadingError) {
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
		body = <OrderBody key={query.data.id} order={query.data} />
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
							{closeLabel}
						</Button>
					</Drawer.Footer>
				</Drawer.Dialog>
			</Drawer.Content>
		</Drawer.Backdrop>
	)
}
