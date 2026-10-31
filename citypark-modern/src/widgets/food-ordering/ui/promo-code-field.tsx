'use client'

import { TextField } from '@/shared/ui'

export function PromoCodeField({
	text,
	onChange,
	onRemove,
	pending,
	error,
	disabled,
}: {
	text: string
	onChange: (value: string) => void
	onRemove: () => void
	pending: boolean
	error?: string
	disabled: boolean
}) {
	return (
		<section aria-label='Промокод' className='bg-surface text-text flex flex-col gap-3 rounded-lg p-4'>
			<TextField
				label='Промокод'
				value={text}
				onChange={(event) => onChange(event.target.value)}
				clearable={!disabled}
				onClear={onRemove}
				error={error}
				autoComplete='off'
				maxLength={256}
				disabled={disabled}
				onKeyDown={(event) => {
					if (event.key === 'Enter') {
						event.preventDefault()
					}
				}}
			/>
			{pending && (
				<p role='status' className='text-text-muted text-sm'>
					Рассчитываем заказ…
				</p>
			)}
		</section>
	)
}
