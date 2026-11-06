'use client'

import { useState } from 'react'
import type { Dish } from '@/entities/dish'
import { AddToCartButton } from '@/features/add-to-cart'
import { formatPrice, useMenuRefresh } from '@/shared/lib'

export function DishControls({ dish, url }: { dish: Dish; url: string }) {
	useMenuRefresh()
	const [variantId, setVariantId] = useState(dish.variants[0].id)
	const [copyStatus, setCopyStatus] = useState<'idle' | 'copied' | 'failed'>('idle')
	const variant = dish.variants.find((item) => item.id === variantId) ?? dish.variants[0]
	async function copyLink() {
		try {
			await navigator.clipboard.writeText(url)
			setCopyStatus('copied')
		} catch {
			setCopyStatus('failed')
		}
	}
	return (
		<div className='flex flex-col items-start gap-4'>
			{dish.variants.length > 1 && (
				<div role='group' aria-label='Размер' className='flex flex-wrap gap-2'>
					{dish.variants.map((item) => (
						<button
							key={item.id}
							type='button'
							aria-pressed={variant.id === item.id}
							onClick={() => setVariantId(item.id)}
							className='rounded-pill border-border aria-pressed:bg-accent aria-pressed:text-on-accent min-h-11 border px-4'
						>
							{item.label}
						</button>
					))}
				</div>
			)}
			<p className='text-text-muted'>{variant.weight}</p>
			<p className='font-display text-2xl'>{variant.price === null ? 'Цена уточняется' : formatPrice(variant.price)}</p>
			{(!dish.available || dish.orderBlocked) && <p className='text-text-muted'>Временно недоступно для заказа</p>}
			<div className='w-full max-w-sm'>
				<AddToCartButton dish={dish} variantId={variant.id} layout='pill' />
			</div>
			<button type='button' onClick={copyLink} className='rounded-pill border-border min-h-11 border px-4'>
				Копировать ссылку
			</button>
			<p role='status'>
				{copyStatus === 'copied'
					? 'Ссылка скопирована'
					: copyStatus === 'failed'
						? 'Не удалось скопировать ссылку. Скопируйте её ниже.'
						: ''}
			</p>
			{copyStatus === 'failed' && (
				<input
					aria-label='Ссылка на блюдо'
					readOnly
					value={url}
					onFocus={(event) => event.target.select()}
					className='border-border bg-surface min-h-11 w-full rounded-sm border px-3'
				/>
			)}
		</div>
	)
}
