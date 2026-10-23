import { Button, Card, FieldError, Form, Input, Label, Skeleton, Spinner, TextField } from '@heroui/react'
import { useQuery } from '@tanstack/react-query'
import { useBlocker } from '@tanstack/react-router'
import { useState, type FormEvent, type ReactNode } from 'react'
import { ApiError } from '../../../shared/api'
import { PageHeader } from '../../../shared/ui/page-header'
import { toastError, toastSuccess } from '../../../shared/ui/toast'
import { hotelPricesQuery, useUpdateHotelPrices } from '../api'
import {
	FEES,
	mapServerFields,
	toForm,
	validateHotelPrices,
	type FeeName,
	type HotelPricesData,
	type HotelPricesErrors,
	type HotelPricesForm,
} from '../model/form'
import { BreakfastList } from './breakfast-list'

const LEAVE_MESSAGE = 'Есть несохранённые изменения. Уйти без сохранения?'

export function HotelPricesPage() {
	const query = useQuery(hotelPricesQuery)
	let content
	if (query.isPending) {
		content = <Skeleton className='h-64 w-full max-w-3xl rounded-xl' aria-label='Загрузка' />
	} else if (query.isError) {
		content = (
			<div role='alert' className='flex flex-col items-start gap-3'>
				<p className='text-danger'>{query.error.message}</p>
				<Button variant='secondary' onPress={() => void query.refetch()}>
					Повторить
				</Button>
			</div>
		)
	} else {
		content = <HotelPricesFormView prices={query.data} />
	}
	return (
		<>
			<PageHeader title='Цены отеля' />
			{content}
		</>
	)
}

function Section({ title, children }: { title: string; children: ReactNode }) {
	return (
		<Card>
			<Card.Header>
				<Card.Title>{title}</Card.Title>
			</Card.Header>
			<Card.Content className='flex flex-col gap-4'>{children}</Card.Content>
		</Card>
	)
}

/** Mount it once per load: the form state starts from `prices` and then follows the saved result. */
function HotelPricesFormView({ prices }: { prices: HotelPricesData }) {
	const update = useUpdateHotelPrices()
	const [baseline, setBaseline] = useState(() => toForm(prices))
	const [form, setForm] = useState<HotelPricesForm>(baseline)
	const [errors, setErrors] = useState<HotelPricesErrors>({})
	const dirty = JSON.stringify(form) !== JSON.stringify(baseline)

	useBlocker({
		shouldBlockFn: () => !window.confirm(LEAVE_MESSAGE),
		enableBeforeUnload: () => dirty,
		disabled: !dirty,
	})

	function fee(name: FeeName, label: string) {
		return (
			<TextField
				key={name}
				name={name}
				value={form[name]}
				onChange={(value) => setForm((current) => ({ ...current, [name]: value }))}
				isInvalid={errors[name] !== undefined}
			>
				<Label>{label}</Label>
				<Input inputMode='decimal' />
				<p className='text-xs text-muted'>Используется в текстах как {`{{${name}}}`}</p>
				<FieldError>{errors[name]}</FieldError>
			</TextField>
		)
	}

	function submit(event: FormEvent<HTMLFormElement>) {
		event.preventDefault()
		const found = validateHotelPrices(form)
		setErrors(found)
		if (Object.keys(found).length > 0) return
		update.mutate(form, {
			onSuccess: (saved) => {
				const next = toForm(saved)
				setBaseline(next)
				setForm(next)
				toastSuccess('Сохранено')
			},
			onError: (error) => {
				if (error instanceof ApiError && error.status === 400) setErrors(mapServerFields(error.fields))
				toastError(error)
			},
		})
	}

	return (
		<Form onSubmit={submit} validationBehavior='aria' className='flex max-w-3xl flex-col gap-6'>
			<Section title='Сборы'>{FEES.map(({ name, label }) => fee(name, label))}</Section>

			<Section title='Меню завтраков'>
				<BreakfastList
					value={form.breakfast}
					errors={errors}
					onChange={(breakfast) => {
						// Row errors are keyed by position, which an add, remove or move changes.
						setErrors((current) =>
							Object.fromEntries(Object.entries(current).filter(([k]) => !k.startsWith('breakfast')))
						)
						setForm((current) => ({ ...current, breakfast }))
					}}
				/>
			</Section>

			<div className='sticky bottom-0 -mx-4 border-t border-separator bg-background px-4 py-3 sm:static sm:mx-0 sm:border-0 sm:p-0'>
				<Button type='submit' variant='primary' isDisabled={!dirty} isPending={update.isPending}>
					{({ isPending }) => (
						<>
							{isPending && <Spinner color='current' size='sm' />}
							Сохранить
						</>
					)}
				</Button>
			</div>
		</Form>
	)
}
