import { Button, Description, FieldError, Form, Input, Label, Spinner, TextField } from '@heroui/react'
import { useBlocker } from '@tanstack/react-router'
import { useState, type FormEvent, type ReactNode } from 'react'
import { ApiError } from '../../../shared/api'
import type { components } from '../../../shared/api/schema'
import { ImageField } from '../../../shared/ui/image-field'
import { RichTextEditor } from '../../../shared/ui/rich-text-editor'
import { toastError, toastSuccess } from '../../../shared/ui/toast'
import { useUpdateRoom } from '../api'
import { mapServerFields, toForm, validateRoom, type RoomErrors, type RoomForm as RoomFormState } from '../model/form'

type Room = components['schemas']['Room']
type TextFieldName = 'name' | 'hotelName' | 'numbers' | 'area' | 'areaDetailed' | 'capacity' | 'hotelNote'

const LEAVE_MESSAGE = 'Есть несохранённые изменения. Уйти без сохранения?'

function Section({ title, children }: { title: string; children: ReactNode }) {
	return (
		<section className='flex flex-col gap-4'>
			<h2 className='text-lg font-semibold'>{title}</h2>
			{children}
		</section>
	)
}

/** Mount it with `key={room.id}`: the form state starts from `room` and then follows the saved result. */
export function RoomForm({ room }: { room: Room }) {
	const update = useUpdateRoom()
	const [baseline, setBaseline] = useState(() => toForm(room))
	const [form, setForm] = useState<RoomFormState>(baseline)
	const [errors, setErrors] = useState<RoomErrors>({})
	const dirty = JSON.stringify(form) !== JSON.stringify(baseline)

	useBlocker({
		shouldBlockFn: () => !window.confirm(LEAVE_MESSAGE),
		enableBeforeUnload: () => dirty,
		disabled: !dirty,
	})

	function set<K extends keyof RoomFormState>(field: K) {
		return (value: RoomFormState[K]) => setForm((current) => ({ ...current, [field]: value }))
	}

	function text(field: TextFieldName, label: string, options: { required?: boolean; help?: string } = {}) {
		return (
			<TextField
				name={field}
				maxLength={100}
				value={form[field]}
				onChange={set(field)}
				isRequired={options.required}
				isInvalid={errors[field] !== undefined}
			>
				<Label>{label}</Label>
				<Input />
				{options.help !== undefined && <Description>{options.help}</Description>}
				<FieldError>{errors[field]}</FieldError>
			</TextField>
		)
	}

	function submit(event: FormEvent<HTMLFormElement>) {
		event.preventDefault()
		const found = validateRoom(form)
		setErrors(found)
		if (Object.keys(found).length > 0) return
		update.mutate(
			{ id: room.id, form },
			{
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
			}
		)
	}

	return (
		<Form onSubmit={submit} validationBehavior='aria' className='flex max-w-2xl flex-col gap-8'>
			<h1 className='text-2xl font-semibold'>{room.name}</h1>
			<dl className='flex flex-wrap gap-x-8 gap-y-1 text-sm text-muted'>
				<div className='flex gap-2'>
					<dt>Тип:</dt>
					<dd>{room.kind}</dd>
				</div>
				<div className='flex gap-2'>
					<dt>Балкон:</dt>
					<dd>{room.balcony ? 'да' : 'нет'}</dd>
				</div>
			</dl>

			<Section title='Основное'>
				{text('name', 'Название', { required: true })}
				{text('hotelName', 'Название в списке на странице отеля', { help: 'Если отличается' })}
				{text('numbers', 'Номера комнат', { required: true, help: 'Например, «№ 204, 304»' })}
				{text('area', 'Площадь', { required: true, help: 'Например, «23 + 7 м²»' })}
				{text('areaDetailed', 'Площадь подробно', { help: 'Для десктопа: «45 м² + балкон 7 м²»' })}
				{text('capacity', 'Вместимость', { help: 'Например, «до 4 гостей»' })}
				{text('hotelNote', 'Строка на странице отеля')}
			</Section>

			<Section title='Цена'>
				<TextField
					name='price'
					inputMode='decimal'
					value={form.price}
					onChange={set('price')}
					isRequired
					isInvalid={errors.price !== undefined}
				>
					<Label>Цена за ночь, ₽</Label>
					<Input />
					<Description>Одна цена на весь год. На главной «от» считается автоматически.</Description>
					<FieldError>{errors.price}</FieldError>
				</TextField>
			</Section>

			<Section title='Описание'>
				<RichTextEditor
					label='Описание'
					profile='short'
					maxLength={1000}
					value={form.description}
					onChange={set('description')}
					error={errors.description}
				/>
			</Section>

			<Section title='Фото'>
				<ImageField label='Фото' aspect={4 / 3} value={form.photo} onChange={set('photo')} errors={errors.photo} />
			</Section>

			<div>
				<Button type='submit' variant='primary' isPending={update.isPending}>
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
