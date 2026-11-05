'use client'

import { useEffect, useRef } from 'react'
import { AddressFields, type AddressFields as Fields } from '@/entities/address'
import { Button, TextField } from '@/shared/ui'
export function AddressEditor({
	values,
	onChange,
	towns,
	suggestionsEnabled,
	errors,
	pending,
	onSave,
	onCancel,
}: {
	values: Fields
	onChange: (values: Fields) => void
	towns: readonly { name: string }[]
	suggestionsEnabled: boolean
	errors?: Record<string, string>
	pending: boolean
	onSave: () => void
	onCancel: () => void
}) {
	const form = useRef<HTMLFormElement>(null)
	useEffect(() => {
		form.current?.querySelector('input')?.focus()
	}, [])
	return (
		<form
			ref={form}
			className='border-border bg-surface flex flex-col gap-4 rounded-lg border p-5'
			onSubmit={(event) => {
				event.preventDefault()
				onSave()
			}}
		>
			<fieldset disabled={pending} className='flex min-w-0 flex-col gap-4'>
				<TextField
					label='Название адреса'
					name='name'
					placeholder='Например, Дом или Работа'
					value={values.name}
					onChange={(event) => onChange({ ...values, name: event.target.value })}
					error={errors?.name}
				/>
				<AddressFields
					values={values}
					onChange={onChange}
					towns={towns}
					suggestionsEnabled={suggestionsEnabled}
					errors={errors}
					onManualEdit={() => {}}
				/>
				<div className='flex flex-wrap gap-3'>
					<Button type='submit'>{pending ? 'Сохраняем…' : 'Сохранить'}</Button>
					<Button variant='outline' onClick={onCancel}>
						Отмена
					</Button>
				</div>
			</fieldset>
		</form>
	)
}
