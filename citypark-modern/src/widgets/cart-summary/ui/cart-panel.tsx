'use client'

import { quoteDelivery, useDeliveryInfo, type Dish } from '@/entities/dish'
import { UnavailableItems } from '@/features/remove-unavailable-items'
import { useSiteSettings } from '@/shared/config'
import { formatPrice, plural } from '@/shared/lib'
import { AppLink, Button } from '@/shared/ui'
import { CartLine } from './cart-line'
import { usePricedCart } from './use-priced-cart'

/** Dark cart column next to the menu (from 1024 px). Below that the CartBar is used instead. */
export function CartPanel({ dishes, checkoutHref = '/checkout' }: { dishes: readonly Dish[]; checkoutHref?: string }) {
	const { items, unavailable, cartCount, subtotal, ready } = usePricedCart(dishes)
	const { restaurant } = useSiteSettings().phones
	const quote = quoteDelivery({ method: 'delivery', zone: 'city', subtotal }, useDeliveryInfo())
	const empty = ready && cartCount === 0

	return (
		<aside
			id='cart'
			aria-label='Корзина'
			className='bg-inverse-bg text-inverse-text hidden w-[340px] shrink-0 flex-col gap-3.5 rounded-xl p-6 lg:sticky lg:top-[160px] lg:flex'
		>
			<div className='flex items-center justify-between'>
				<h2 className='heading text-2xl'>Корзина</h2>
				{cartCount > 0 && (
					<span className='text-inverse-muted text-sm'>
						{cartCount} {plural(cartCount, ['блюдо', 'блюда', 'блюд'])}
					</span>
				)}
			</div>

			{empty ? (
				<p className='text-inverse-muted text-[15px]'>Пока пусто. Добавьте блюда кнопкой «+» на карточках.</p>
			) : (
				<>
					<ul>
						{items.map((item) => (
							<CartLine key={`${item.dish.id}:${item.variant.id}`} item={item} />
						))}
					</ul>
					<UnavailableItems lines={unavailable} />
					<div className='flex justify-between text-[15px]'>
						<span className='text-inverse-muted'>Доставка</span>
						<span className='text-accent font-semibold'>{quote.label}</span>
					</div>
					<div className='flex items-baseline justify-between'>
						<span>Итого</span>
						<span className='font-display text-[30px] font-medium'>{formatPrice(subtotal)}</span>
					</div>
					{unavailable.length > 0 ? (
						<Button disabled className='min-h-[54px]'>
							Оформить заказ
						</Button>
					) : (
						<Button href={checkoutHref} className='min-h-[54px]'>
							Оформить заказ
						</Button>
					)}
				</>
			)}

			<p className='text-inverse-muted text-center text-[13px]'>
				{empty ? 'Заказ по телефону: ' : 'или '}
				<AppLink href={restaurant.href} className='tap-area text-inverse-text'>
					{restaurant.display}
				</AppLink>
			</p>
		</aside>
	)
}
