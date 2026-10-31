'use client'

import { Button, TextField } from '@/shared/ui'

export function PromoCodeField({
	text,
	onChange,
	onApply,
	onRemove,
	pending,
	error,
	disabled,
}: {
	text: string
	onChange: (value: string) => void
	onApply: () => void
	onRemove: () => void
	pending: boolean
	error?: string
	disabled: boolean
}) {
	return (
		<section aria-label='Промокод' className='bg-surface text-text flex flex-col gap-2 rounded-lg p-4'>
			<TextField
				label='Промокод'
				value={text}
				onChange={(event) => onChange(event.target.value)}
				error={error}
				autoComplete='off'
				maxLength={256}
				disabled={disabled}
				onKeyDown={(event) => {
					if (event.key === 'Enter') {
						event.preventDefault()
						if (!disabled) onApply()
					}
				}}
			/>
			<div className='flex flex-wrap gap-2'>
				<Button disabled={disabled} onClick={onApply}>
					{text.trim() ? 'Применить' : 'Пересчитать'}
				</Button>
				{text && (
					<Button variant='outline' disabled={disabled} onClick={onRemove}>
						Удалить
					</Button>
				)}
			</div>
			{pending && (
				<p role='status' className='text-text-muted text-sm'>
					Рассчитываем заказ…
				</p>
			)}
		</section>
	)
}
