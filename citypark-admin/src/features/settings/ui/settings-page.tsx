import { Button, Card, FieldError, Form, Input, Label, Skeleton, Spinner, TextField } from '@heroui/react'
import { useQuery } from '@tanstack/react-query'
import { useBlocker } from '@tanstack/react-router'
import { useState, type FormEvent, type ReactNode } from 'react'
import { ApiError } from '../../../shared/api'
import { PageHeader } from '../../../shared/ui/page-header'
import { toastError, toastSuccess } from '../../../shared/ui/toast'
import { settingsQuery, useUpdateSettings } from '../api'
import {
	PHONES,
	addressPreview,
	mapServerFields,
	toForm,
	validateSettings,
	type SettingsData,
	type SettingsErrors,
	type SettingsForm,
} from '../model/form'
import { PhoneFields } from './phone-fields'
import { SocialList } from './social-list'

const LEAVE_MESSAGE = 'Есть несохранённые изменения. Уйти без сохранения?'

export function SettingsPage() {
	const query = useQuery(settingsQuery)
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
		content = <SettingsFormView settings={query.data} />
	}
	return (
		<>
			<PageHeader title='Настройки сайта' />
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

/** Mount it once per load: the form state starts from `settings` and then follows the saved result. */
function SettingsFormView({ settings }: { settings: SettingsData }) {
	const update = useUpdateSettings()
	const [baseline, setBaseline] = useState(() => toForm(settings))
	const [form, setForm] = useState<SettingsForm>(baseline)
	const [errors, setErrors] = useState<SettingsErrors>({})
	const dirty = JSON.stringify(form) !== JSON.stringify(baseline)

	useBlocker({
		shouldBlockFn: () => !window.confirm(LEAVE_MESSAGE),
		enableBeforeUnload: () => dirty,
		disabled: !dirty,
	})

	function text(key: string, label: string, value: string, onChange: (value: string) => void, inputMode?: 'numeric') {
		return (
			<TextField
				name={key}
				value={value}
				onChange={onChange}
				isInvalid={errors[key] !== undefined}
				className='min-w-48 flex-1'
			>
				<Label>{label}</Label>
				<Input inputMode={inputMode} />
				<FieldError>{errors[key]}</FieldError>
			</TextField>
		)
	}

	function setAddress(patch: Partial<SettingsForm['address']>) {
		setForm((current) => ({ ...current, address: { ...current.address, ...patch } }))
	}

	function setCompany(patch: Partial<SettingsForm['company']>) {
		setForm((current) => ({ ...current, company: { ...current.company, ...patch } }))
	}

	function submit(event: FormEvent<HTMLFormElement>) {
		event.preventDefault()
		const found = validateSettings(form)
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
			<Section title='Телефоны'>
				{PHONES.map(({ name, title }) => (
					<PhoneFields
						key={name}
						name={name}
						title={title}
						value={form.phones[name]}
						errors={errors}
						onChange={(value) => {
							setForm((current) => ({ ...current, phones: { ...current.phones, [name]: value } }))
						}}
					/>
				))}
			</Section>

			<Section title='Адрес'>
				<div className='flex flex-wrap gap-4'>
					{text('address.postcode', 'Индекс', form.address.postcode, (postcode) => setAddress({ postcode }), 'numeric')}
					{text('address.city', 'Город', form.address.city, (city) => setAddress({ city }))}
				</div>
				{text('address.street', 'Улица и дом', form.address.street, (street) => setAddress({ street }))}
				<p className='text-sm text-muted'>
					На сайте: <span data-testid='address-preview'>{addressPreview(form.address)}</span>
				</p>
			</Section>

			<Section title='E-mail'>
				{text('email', 'E-mail', form.email, (email) => setForm((current) => ({ ...current, email })))}
			</Section>

			<Section title='Юрлицо'>
				{text('company.name', 'Название', form.company.name, (name) => setCompany({ name }))}
				<div className='flex flex-wrap gap-4'>
					{text('company.inn', 'ИНН', form.company.inn, (inn) => setCompany({ inn }), 'numeric')}
					{text('company.ogrn', 'ОГРН', form.company.ogrn, (ogrn) => setCompany({ ogrn }), 'numeric')}
				</div>
			</Section>

			<Section title='Соцсети'>
				<SocialList
					value={form.social}
					errors={errors}
					onChange={(social) => {
						// Row errors are keyed by position, which an add, remove or move changes.
						setErrors((current) => Object.fromEntries(Object.entries(current).filter(([k]) => !k.startsWith('social'))))
						setForm((current) => ({ ...current, social }))
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
