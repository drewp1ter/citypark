import { Button, FieldError, Form, Input, Label, Spinner, TextField } from '@heroui/react'
import { useRef, useState, type FormEvent } from 'react'
import { ApiError } from '../../../shared/api'
import type { components } from '../../../shared/api/schema'
import { formatDateTime } from '../../../shared/lib/format'
import { toastError, toastSuccess } from '../../../shared/ui/toast'
import { useSaveDeliverySettings } from '../api'
import {
	mapServerFields,
	toForm,
	validateDeliverySettings,
	type DeliverySettingsErrors,
	type DeliverySettingsForm,
} from '../model/form'
export function DeliverySettingsFormView({ settings }: { settings: components['schemas']['DeliverySettings'] }) {
	const save = useSaveDeliverySettings()
	const [baseline, setBaseline] = useState(settings)
	const [form, setForm] = useState(() => toForm(settings))
	const [errors, setErrors] = useState<DeliverySettingsErrors>({})
	const locked = useRef(false)
	const original = toForm(baseline)
	const dirty = (Object.keys(form) as (keyof DeliverySettingsForm)[]).some((key) => form[key] !== original[key])
	// Adopt background updates only while pristine; never remount a dirty form.
	if (settings !== baseline && !dirty && !save.isPending) {
		setBaseline(settings)
		setForm(toForm(settings))
	}
	function submit(event: FormEvent<HTMLFormElement>) {
		event.preventDefault()
		if (!dirty || locked.current) return
		const found = validateDeliverySettings(form)
		setErrors(found)
		if (Object.keys(found).length) return
		locked.current = true
		save.mutate(form, {
			onSuccess: (saved) => {
				setBaseline(saved)
				setForm(toForm(saved))
				setErrors({})
				toastSuccess('Сохранено')
			},
			onError: (error) => {
				if (error instanceof ApiError) setErrors(mapServerFields(error.fields))
				toastError(error)
			},
			onSettled: () => {
				locked.current = false
			},
		})
	}
	function field(name: keyof DeliverySettingsForm, label: string, numeric = false) {
		return (
			<TextField
				name={name}
				value={form[name]}
				onChange={(value) => setForm((current) => ({ ...current, [name]: value }))}
				isDisabled={save.isPending}
				isInvalid={errors[name] !== undefined}
			>
				<Label>{label}</Label>
				<Input inputMode={numeric ? 'numeric' : 'text'} />
				<FieldError>{errors[name]}</FieldError>
			</TextField>
		)
	}
	return (
		<section aria-labelledby='delivery-settings-title' className='mb-6 rounded-xl border border-border bg-surface p-4'>
			<h2 id='delivery-settings-title' className='text-lg font-semibold'>
				Информация о доставке
			</h2>
			<p className='mb-4 text-sm text-muted'>
				Время приготовления и стоимость доставки в СОТ и ДНТ показываются на сайте. Цены по городам и порог бесплатной
				доставки редактируются ниже.
			</p>
			<Form onSubmit={submit} validationBehavior='aria' className='flex flex-col gap-4' aria-busy={save.isPending}>
				<div className='grid gap-4 sm:grid-cols-2'>
					{field('cooking', 'Время приготовления')}
					{field('outskirtsLabel', 'Зона доставки')}
					<p className='text-sm text-muted sm:col-span-2'>Например: ~30 мин</p>
					{field('outskirtsMin', 'Цена от, ₽', true)}
					{field('outskirtsMax', 'Цена до, ₽', true)}
				</div>
				<p className='text-sm text-muted'>
					Информационный диапазон. Стоимость заказа рассчитывается по тарифу выбранного города.
				</p>
				<div>
					<Button type='submit' variant='primary' isDisabled={!dirty || save.isPending} isPending={save.isPending}>
						{save.isPending && <Spinner color='current' size='sm' />}Сохранить
					</Button>
				</div>
			</Form>
			<p className='mt-3 text-sm text-muted'>Изменено {formatDateTime(baseline.updatedAt)}</p>
		</section>
	)
}
