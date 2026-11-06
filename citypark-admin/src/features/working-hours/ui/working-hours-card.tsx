import { Button, FieldError, Form, Input, Label, Skeleton, Spinner, TextField } from '@heroui/react'
import { useQuery } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'
import { ApiError } from '../../../shared/api'
import { formatDateTime } from '../../../shared/lib/format'
import { toastError, toastSuccess } from '../../../shared/ui/toast'
import { useSaveWorkingHours, workingHoursQuery } from '../api'
import {
	mapServerFields,
	toForm,
	validateWorkingHours,
	type WorkingHoursErrors,
	type WorkingHoursForm,
} from '../model/form'

const CARD = 'mb-6 rounded-xl border border-border bg-surface p-4'

type Hours = { start: string; end: string; updatedAt: string }

export function WorkingHoursCard() {
	const query = useQuery(workingHoursQuery)
	if (query.isPending) {
		return <Skeleton className='mb-6 h-40 w-full rounded-xl' aria-label='Загрузка' />
	}
	if (query.isError) {
		return (
			<div role='alert' className={`${CARD} flex flex-col items-start gap-3`}>
				<p className='text-danger'>{query.error.message}</p>
				<Button variant='secondary' onPress={() => void query.refetch()}>
					Повторить
				</Button>
			</div>
		)
	}
	// Mount the form once per loaded value: it starts from the server value and then follows the saved result.
	return <WorkingHoursFormView key={query.data.updatedAt} hours={query.data} />
}

function WorkingHoursFormView({ hours }: { hours: Hours }) {
	const save = useSaveWorkingHours()
	const [form, setForm] = useState<WorkingHoursForm>(() => toForm(hours))
	const [errors, setErrors] = useState<WorkingHoursErrors>({})
	const dirty = form.start !== hours.start || form.end !== hours.end

	function submit(event: FormEvent<HTMLFormElement>) {
		event.preventDefault()
		const found = validateWorkingHours(form)
		setErrors(found)
		if (Object.keys(found).length > 0) return
		save.mutate(form, {
			onSuccess: () => toastSuccess('Сохранено'),
			onError: (error) => {
				if (error instanceof ApiError && error.status === 400) setErrors(mapServerFields(error.fields))
				toastError(error)
			},
		})
	}

	function field(name: keyof WorkingHoursForm, label: string) {
		return (
			<TextField
				name={name}
				value={form[name]}
				onChange={(value) => setForm((current) => ({ ...current, [name]: value }))}
				isInvalid={errors[name] !== undefined}
				className='w-full sm:w-40'
			>
				<Label>{label}</Label>
				<Input inputMode='text' placeholder='09:00' />
				<FieldError>{errors[name]}</FieldError>
			</TextField>
		)
	}

	return (
		<section className={CARD} aria-labelledby='working-hours-title'>
			<h2 id='working-hours-title' className='text-lg font-semibold'>
				Часы приёма заказов
			</h2>
			<p className='mb-4 text-sm text-muted'>
				Сайт принимает заказы и показывает эти часы. Заказ «ко времени» можно выбрать только внутри них.
			</p>
			<Form onSubmit={submit} validationBehavior='aria' className='flex flex-col gap-4'>
				<div className='flex flex-col gap-4 sm:flex-row'>
					{field('start', 'С')}
					{field('end', 'До')}
				</div>
				<div>
					<Button type='submit' variant='primary' isDisabled={!dirty} isPending={save.isPending}>
						{({ isPending }) => (
							<>
								{isPending && <Spinner color='current' size='sm' />}
								Сохранить
							</>
						)}
					</Button>
				</div>
			</Form>
			<p className='mt-3 text-sm text-muted'>Изменено {formatDateTime(hours.updatedAt)}</p>
		</section>
	)
}
