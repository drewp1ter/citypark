import type { Order } from '../model/types'
import { formatOrderPrice, orderPayable } from '../model/money'

/** Render only persisted order prices: campaigns and the current menu cannot change history. */
export function OrderBreakdown({ order }: { order: Order }) {
	const promo = Boolean(order.promoCode)
	const price = (value: number) => formatOrderPrice(order, value)
	if (!promo)
		return (
			<>
				<ul className='flex flex-col gap-2'>
					{order.items.map((item, index) => (
						<li key={`${item.productId}-${index}`} className='flex justify-between gap-3'>
							<span>
								{item.name} × {item.amount}
							</span>
							<span>{price(item.offeredPrice * item.amount)}</span>
						</li>
					))}
				</ul>
				<p>Товары: {price(order.total)}</p>
				<p>Доставка: {price(order.deliveryPrice)}</p>
				<p className='font-semibold'>Итого: {price(orderPayable(order))}</p>
			</>
		)
	return (
		<>
			<ul className='flex flex-col gap-2'>
				{order.items.map((item, index) => {
					const original = Math.round(item.offeredPrice * 100) * item.amount
					return (
						<li key={`${item.productId}-${index}`} className='flex flex-wrap justify-between gap-3'>
							<span className='min-w-0 break-words'>
								{item.name} × {item.amount}
							</span>
							<div className='ml-auto text-right'>
								<p>{price(original / 100)}</p>
								<p>Скидка: {price(0 - item.discount)}</p>
								<p>Со скидкой: {price((original - Math.round(item.discount * 100)) / 100)}</p>
							</div>
						</li>
					)
				})}
			</ul>
			<dl aria-label='Итоги заказа' className='flex flex-col gap-3'>
				<div className='flex flex-wrap justify-between gap-2'>
					<dt>Блюда до скидки:</dt>
					<dd>{price(order.subtotal)}</dd>
				</div>
				<div className='flex flex-wrap justify-between gap-2'>
					<dt>Промокод:</dt>
					<dd className='break-all'>{order.promoCode}</dd>
				</div>
				<div className='flex flex-wrap justify-between gap-2'>
					<dt>Скидка по промокоду:</dt>
					<dd>{price(0 - order.discount)}</dd>
				</div>
				<div className='flex flex-wrap justify-between gap-2'>
					<dt>Блюда со скидкой:</dt>
					<dd>{price(order.total)}</dd>
				</div>
				<div className='flex flex-wrap justify-between gap-2'>
					<dt>Доставка:</dt>
					<dd>{price(order.deliveryPrice)}</dd>
				</div>
				<div className='flex flex-wrap justify-between gap-2 font-semibold'>
					<dt>Итого:</dt>
					<dd>{price(orderPayable(order))}</dd>
				</div>
			</dl>
		</>
	)
}
