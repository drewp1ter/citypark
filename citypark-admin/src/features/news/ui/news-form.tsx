import {
	Button,
	Calendar,
	DateField,
	DatePicker,
	Description,
	FieldError,
	Form,
	I18nProvider,
	Input,
	Label,
	Spinner,
	TextArea,
	TextField,
} from '@heroui/react'
import { parseDate } from '@internationalized/date'
import { useNavigate, useBlocker } from '@tanstack/react-router'
import { useRef, useState, type FormEvent } from 'react'
import { ApiError } from '../../../shared/api'
import { ImageField } from '../../../shared/ui/image-field'
import { RichTextEditor } from '../../../shared/ui/rich-text-editor'
import { toastError, toastSuccess } from '../../../shared/ui/toast'
import { useCreateNews, useUpdateNews } from '../api'
import {
	MAX_BODY,
	MAX_EXCERPT,
	MAX_SHORT_TITLE,
	MAX_TITLE,
	mapServerFields,
	validateNews,
	type NewsErrors,
	type NewsForm as NewsFormState,
} from '../model/form'
import { DeleteNewsDialog } from './delete-news-dialog'

const LEAVE_MESSAGE = 'Есть несохранённые изменения. Уйти без сохранения?'
const SLUG_TAKEN = 'Такой адрес уже занят'

type Props = {
	/** Starting values: an empty form for a new item, the stored item for an edit. */
	initial: NewsFormState
	/** The stored item's id; `undefined` for a new item. */
	id?: number
}

/** Mount it with `key={slug}`: the form state starts from `initial` and then follows the saved result. */
export function NewsForm({ initial, id }: Props) {
	const create = useCreateNews()
	const update = useUpdateNews()
	const navigate = useNavigate()
	const [baseline, setBaseline] = useState(initial)
	const [form, setForm] = useState<NewsFormState>(initial)
	const [errors, setErrors] = useState<NewsErrors>({})
	const [confirmDelete, setConfirmDelete] = useState(false)
	// Set just before our own navigation (after a save or a delete): the blocker must not ask then.
	const leaving = useRef(false)
	const dirty = JSON.stringify(form) !== JSON.stringify(baseline)
	const pending = create.isPending || update.isPending

	useBlocker({
		shouldBlockFn: () => !leaving.current && !window.confirm(LEAVE_MESSAGE),
		enableBeforeUnload: () => dirty,
		disabled: !dirty,
	})

	function set<K extends keyof NewsFormState>(field: K) {
		return (value: NewsFormState[K]) => setForm((current) => ({ ...current, [field]: value }))
	}

	function fail(error: Error) {
		if (error instanceof ApiError) {
			if (error.status === 409) setErrors({ slug: SLUG_TAKEN })
			else if (error.status === 400) setErrors(mapServerFields(error.fields))
		}
		toastError(error)
	}

	function submit(event: FormEvent<HTMLFormElement>) {
		event.preventDefault()
		const found = validateNews(form)
		setErrors(found)
		if (Object.keys(found).length > 0) return
		if (id === undefined) {
			create.mutate(form, {
				onSuccess: (saved) => {
					// The form is done: the edit page of the new item takes over.
					leaving.current = true
					toastSuccess('Новость опубликована')
					void navigate({ to: '/content/news/$slug', params: { slug: saved.slug }, replace: true })
				},
				onError: fail,
			})
			return
		}
		update.mutate(
			{ id, form },
			{
				onSuccess: (saved) => {
					const next = { ...form, slug: saved.slug, body: saved.body }
					setBaseline(next)
					setForm(next)
					toastSuccess('Сохранено')
					if (saved.slug !== initial.slug) {
						leaving.current = true
						void navigate({ to: '/content/news/$slug', params: { slug: saved.slug }, replace: true })
					}
				},
				onError: fail,
			}
		)
	}

	return (
		<Form onSubmit={submit} validationBehavior='aria' className='flex max-w-2xl flex-col gap-6'>
			<TextField
				name='title'
				maxLength={MAX_TITLE}
				value={form.title}
				onChange={set('title')}
				isRequired
				isInvalid={errors.title !== undefined}
			>
				<Label>Заголовок</Label>
				<Input />
				<FieldError>{errors.title}</FieldError>
			</TextField>

			<TextField
				name='shortTitle'
				maxLength={MAX_SHORT_TITLE}
				value={form.shortTitle}
				onChange={set('shortTitle')}
				isInvalid={errors.shortTitle !== undefined}
			>
				<Label>Короткий заголовок</Label>
				<Input />
				<Description>Для главной и мобильной версии, если заголовок длинный</Description>
				<FieldError>{errors.shortTitle}</FieldError>
			</TextField>

			<TextField
				name='excerpt'
				maxLength={MAX_EXCERPT}
				value={form.excerpt}
				onChange={set('excerpt')}
				isInvalid={errors.excerpt !== undefined}
			>
				<Label>Анонс</Label>
				<TextArea rows={3} />
				<FieldError>{errors.excerpt}</FieldError>
			</TextField>

			<I18nProvider locale='ru-RU'>
				<DatePicker
					name='date'
					className='w-full sm:w-64'
					value={form.date === '' ? null : parseDate(form.date)}
					onChange={(value) => set('date')(value === null ? '' : value.toString())}
					isRequired
					isInvalid={errors.date !== undefined}
				>
					<Label>Дата</Label>
					<DateField.Group>
						<DateField.Input>{(segment) => <DateField.Segment segment={segment} />}</DateField.Input>
						<DateField.Suffix>
							<DatePicker.Trigger>
								<DatePicker.TriggerIndicator />
							</DatePicker.Trigger>
						</DateField.Suffix>
					</DateField.Group>
					<FieldError>{errors.date}</FieldError>
					<DatePicker.Popover>
						<Calendar aria-label='Дата новости'>
							<Calendar.Header>
								<Calendar.Heading />
								<Calendar.NavButton slot='previous' />
								<Calendar.NavButton slot='next' />
							</Calendar.Header>
							<Calendar.Grid>
								<Calendar.GridHeader>{(day) => <Calendar.HeaderCell>{day}</Calendar.HeaderCell>}</Calendar.GridHeader>
								<Calendar.GridBody>{(date) => <Calendar.Cell date={date} />}</Calendar.GridBody>
							</Calendar.Grid>
						</Calendar>
					</DatePicker.Popover>
				</DatePicker>
			</I18nProvider>

			<TextField
				name='slug'
				maxLength={100}
				value={form.slug}
				onChange={set('slug')}
				isInvalid={errors.slug !== undefined}
			>
				<Label>Адрес на сайте</Label>
				<Input />
				{id === undefined && <Description>Оставьте пустым — заполнится из заголовка</Description>}
				{id !== undefined && (
					<Description>Если изменить адрес, старые ссылки на новость перестанут работать</Description>
				)}
				<FieldError>{errors.slug}</FieldError>
			</TextField>

			<ImageField
				label='Фото'
				aspect={16 / 9}
				allowEmpty
				value={form.photo}
				onChange={set('photo')}
				errors={errors.photo}
			/>

			<RichTextEditor
				label='Текст'
				profile='article'
				maxLength={MAX_BODY}
				value={form.body}
				onChange={set('body')}
				error={errors.body}
			/>

			<div className='flex flex-wrap items-center gap-3'>
				<Button type='submit' variant='primary' isPending={pending}>
					{({ isPending }) => (
						<>
							{isPending && <Spinner color='current' size='sm' />}
							{id === undefined ? 'Опубликовать' : 'Сохранить'}
						</>
					)}
				</Button>
				{id !== undefined && (
					<Button variant='danger-soft' onPress={() => setConfirmDelete(true)}>
						Удалить
					</Button>
				)}
			</div>
			{id !== undefined && (
				<DeleteNewsDialog
					item={confirmDelete ? { id, title: initial.title } : null}
					onClose={() => setConfirmDelete(false)}
					onDeleted={() => {
						leaving.current = true
						void navigate({ to: '/content/news', replace: true })
					}}
				/>
			)}
		</Form>
	)
}
