'use client'

import { cartActions } from '@/entities/cart'
import type { UnavailableLine } from '@/entities/dish'

export function UnavailableItems({ lines, disabled }: { lines: readonly UnavailableLine[]; disabled?: boolean }) {
	if (lines.length === 0) return null

	return (
		<section className='bg-surface text-text rounded-sm p-3 text-sm'>
			<p role='alert' className='text-danger'>
				Удалите недоступные блюда, чтобы продолжить.
			</p>
			<ul>
				{lines.map((line) => {
					const name = line.name ?? 'Недоступное блюдо'
					return (
						<li
							key={`${line.dishId}:${line.variantId}`}
							className='border-border flex items-center justify-between gap-2 border-b'
						>
							<span>
								{name} × {line.quantity}
							</span>
							<button
								type='button'
								aria-label={`Удалить из корзины: ${name}`}
								className='min-h-11 shrink-0 cursor-pointer underline disabled:opacity-50'
								disabled={disabled}
								onClick={() => cartActions.remove(line)}
							>
								Удалить
							</button>
						</li>
					)
				})}
			</ul>
		</section>
	)
}
