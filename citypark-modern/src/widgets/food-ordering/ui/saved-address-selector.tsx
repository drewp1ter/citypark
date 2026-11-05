'use client'

import { AppLink, Select } from '@/shared/ui'
import type { CheckoutAddressEntry } from '../model/use-checkout-address'

type Props = {
	entries: readonly CheckoutAddressEntry[]
	selected: string
	towns: readonly { name: string }[]
	onSelect: (key: string) => void
	mini: boolean
}
export function SavedAddressSelector({ entries, selected, towns, onSelect, mini }: Props) {
	return (
		<div className='flex flex-col gap-2'>
			<Select
				label='Сохранённые адреса'
				name='savedAddress'
				value={selected}
				onValueChange={onSelect}
				options={[
					{ value: '', label: 'Другой адрес' },
					...entries.map((entry) => {
						const unavailable = !towns.some((town) => town.name === entry.city)
						const reason = entry.needsReview
							? 'Уточните адрес в разделе «Мои адреса»'
							: unavailable
								? 'Доставка недоступна'
								: ''
						return {
							value: entry.key,
							label: `${entry.name || `${entry.city}, ${entry.street}`}${reason ? ` — ${reason}` : ''}`,
							disabled: !!reason,
						}
					}),
				]}
			/>
			<button type='button' onClick={() => onSelect('')} className='tap-area self-start underline'>
				Другой адрес
			</button>
			<AppLink href={mini ? '/mini/addresses' : '/account/addresses'} className='tap-area self-start underline'>
				Мои адреса
			</AppLink>
		</div>
	)
}
