'use client'

import { Select, TextField } from '@/shared/ui'
import type { AddressFields as Fields } from '../model/types'
import { AddressField } from './address-field'

export type AddressFields = Fields

type Props = {
	values: Fields
	onChange: (values: Fields) => void
	towns: readonly { name: string }[]
	suggestionsEnabled: boolean
	errors?: Partial<Record<keyof Fields, string>>
	onManualEdit: () => void
}

export function AddressFields({ values, onChange, towns, suggestionsEnabled, errors = {}, onManualEdit }: Props) {
	const names = [...new Set(towns.map((town) => town.name))]
	if (values.city && !names.includes(values.city)) names.unshift(values.city)
	const change = (field: keyof Fields, value: string) => onChange({ ...values, [field]: value })
	return (
		<>
			<Select
				label='Населённый пункт'
				name='city'
				value={values.city}
				options={[
					{ value: '', label: 'Выберите населённый пункт' },
					...names.map((name) => ({ value: name, label: name })),
				]}
				onValueChange={(value) => {
					onManualEdit()
					onChange({ ...values, city: value, street: '' })
				}}
				error={errors.city}
			/>
			<AddressField
				label='Улица и дом'
				name='street'
				town={values.city}
				suggestions={suggestionsEnabled}
				value={values.street}
				onValueChange={(value) => change('street', value)}
				onManualEdit={onManualEdit}
				error={errors.street}
			/>
			<TextField
				label='Квартира / офис'
				name='apartment'
				value={values.apartment}
				onChange={(event) => {
					onManualEdit()
					change('apartment', event.target.value)
				}}
				error={errors.apartment}
			/>
			<TextField
				label='Подъезд, этаж'
				name='entrance'
				value={values.entrance}
				onChange={(event) => {
					onManualEdit()
					change('entrance', event.target.value)
				}}
				error={errors.entrance}
			/>
		</>
	)
}
