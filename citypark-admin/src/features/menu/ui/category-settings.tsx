import { Button, FieldError, Form, Input, Label, TextField } from '@heroui/react'
import { useState, type FormEvent } from 'react'
import { ApiError } from '../../../shared/api'
import { toastError } from '../../../shared/ui/toast'
import { useSaveCategoryHours } from '../api'
import { categoryDays, WEEKDAYS } from '../model/category-hours'
import type { ApiCategory } from '../model/group'

export function CategorySettings({ category }: { category: ApiCategory }) {
	const save = useSaveCategoryHours(category.id)
	const [startTime, setStartTime] = useState(category.startTime)
	const [endTime, setEndTime] = useState(category.endTime)
	const [days, setDays] = useState(() => categoryDays(category.salesDays))
	const [errors, setErrors] = useState<Record<string, string>>({})
	function submit(event: FormEvent<HTMLFormElement>) {
		event.preventDefault()
		setErrors({})
		save.mutate(
			{ startTime, endTime, days: days.every(Boolean) ? '' : days.map((day) => (day ? '1' : '0')).join('') },
			{
				onError: (error) => {
					if (error instanceof ApiError) setErrors(error.fields ?? {})
					toastError(error)
				},
			}
		)
	}
	return (
		<section className='rounded-xl border border-border bg-surface p-4' aria-label='Настройки категории'>
			<h2 className='mb-2 text-lg font-semibold'>{category.name}</h2>
			{category.description && <p className='mb-3 text-sm whitespace-pre-line text-muted'>{category.description}</p>}
			<Form onSubmit={submit} validationBehavior='aria' className='flex flex-col gap-3'>
				<p className='text-sm text-muted'>
					Время заказа категории. Оставьте оба поля пустыми для заказа в течение всего рабочего дня. Часовой пояс
					ресторана.
				</p>
				<div className='flex flex-col gap-3 sm:flex-row'>
					<TextField name='startTime' value={startTime} onChange={setStartTime} isInvalid={!!errors.startTime}>
						<Label>Начало заказа</Label>
						<Input placeholder='09:00' inputMode='numeric' />
						<FieldError>{errors.startTime}</FieldError>
					</TextField>
					<TextField name='endTime' value={endTime} onChange={setEndTime} isInvalid={!!errors.endTime}>
						<Label>Окончание заказа</Label>
						<Input placeholder='21:00' inputMode='numeric' />
						<FieldError>{errors.endTime}</FieldError>
					</TextField>
				</div>
				<fieldset aria-describedby={errors.days ? 'category-days-error' : undefined}>
					<legend className='mb-2'>Дни заказа</legend>
					<div className='flex flex-wrap gap-2'>
						{WEEKDAYS.map((day, index) => (
							<label key={day} className='flex min-h-11 items-center gap-2 rounded-lg border border-border px-3'>
								<input
									type='checkbox'
									checked={days[index]}
									onChange={(event) =>
										setDays((current) => current.map((value, i) => (i === index ? event.target.checked : value)))
									}
								/>
								{day}
							</label>
						))}
					</div>
					{errors.days && (
						<p id='category-days-error' className='text-sm text-danger' role='alert'>
							{errors.days}
						</p>
					)}
				</fieldset>
				<Button type='submit' isPending={save.isPending} className='self-start'>
					Сохранить расписание
				</Button>
			</Form>
		</section>
	)
}
