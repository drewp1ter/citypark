import { Button, FieldError, Form, Input, Label, TextField } from '@heroui/react'
import { useQuery } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'
import { ApiError } from '../../../shared/api'
import { toastError } from '../../../shared/ui/toast'
import { categoriesQuery, productsQuery, useSaveCampaign } from '../api'
import { formBody, initialForm, type Campaign } from '../model/form'
import { TargetPicker } from './target-picker'

export function PromoCodeForm({ campaign, onClose }: { campaign?: Campaign; onClose: () => void }) {
	const [form, setForm] = useState(() => initialForm(campaign))
	const [errors, setErrors] = useState<Record<string, string>>({})
	const categories = useQuery(categoriesQuery)
	const products = useQuery(productsQuery)
	const save = useSaveCampaign(campaign?.id)
	function submit(e: FormEvent<HTMLFormElement>) {
		e.preventDefault()
		const result = formBody(form)
		setErrors(result.errors)
		if (Object.keys(result.errors).length) return
		save.mutate(result.body, {
			onSuccess: onClose,
			onError: (error) => {
				if (error instanceof ApiError) setErrors(error.fields)
				toastError(error)
			},
		})
	}
	return (
		<section aria-label='Редактор промокода' className='mb-6 rounded-xl border border-border bg-surface p-4'>
			<h2 className='mb-4 text-xl font-semibold'>{campaign ? `Промокод ${campaign.code}` : 'Новый промокод'}</h2>
			<Form onSubmit={submit} validationBehavior='aria' className='flex flex-col gap-4'>
				<fieldset disabled={save.isPending} className='flex min-w-0 flex-col gap-4'>
					<div className='grid gap-4 sm:grid-cols-2'>
						<TextField
							name='code'
							value={form.code}
							onChange={(code) => setForm({ ...form, code })}
							isReadOnly={!!campaign}
							isInvalid={!!errors.code}
						>
							<Label>Промокод</Label>
							<Input autoComplete='off' />
							<FieldError>{errors.code}</FieldError>
						</TextField>
						<TextField
							name='percentage'
							value={form.percentage}
							onChange={(percentage) => setForm({ ...form, percentage })}
							isInvalid={!!errors.percentage}
						>
							<Label>Процент скидки</Label>
							<Input inputMode='numeric' />
							<FieldError>{errors.percentage}</FieldError>
						</TextField>
						<TextField
							name='minimumEligibleSubtotal'
							value={form.minimumEligibleSubtotal}
							onChange={(minimumEligibleSubtotal) => setForm({ ...form, minimumEligibleSubtotal })}
							isInvalid={!!errors.minimumEligibleSubtotal}
						>
							<Label>Минимальная сумма подходящих блюд</Label>
							<Input inputMode='decimal' />
							<FieldError>{errors.minimumEligibleSubtotal}</FieldError>
						</TextField>
						<TextField
							name='maximumDiscount'
							value={form.maximumDiscount}
							onChange={(maximumDiscount) => setForm({ ...form, maximumDiscount })}
							isInvalid={!!errors.maximumDiscount}
						>
							<Label>Максимальная скидка</Label>
							<Input inputMode='decimal' />
							<FieldError>{errors.maximumDiscount}</FieldError>
						</TextField>
						<TextField
							name='startsAt'
							value={form.startsAt}
							onChange={(startsAt) => setForm({ ...form, startsAt })}
							isInvalid={!!errors.startsAt}
						>
							<Label>Начало</Label>
							<Input type='datetime-local' />
							<FieldError>{errors.startsAt}</FieldError>
						</TextField>
						<TextField
							name='endsAt'
							value={form.endsAt}
							onChange={(endsAt) => setForm({ ...form, endsAt })}
							isInvalid={!!errors.endsAt}
						>
							<Label>Окончание</Label>
							<Input type='datetime-local' />
							<FieldError>{errors.endsAt}</FieldError>
						</TextField>
					</div>
					<p className='text-sm text-muted'>
						Время — московское (Europe/Moscow). Пустые даты: начало сразу, без окончания. Суммы указаны в рублях.
						Минимальная сумма учитывает только подходящие блюда. Пустые суммы — без ограничения.
					</p>
					<label className='flex min-h-11 items-center gap-2'>
						<input
							type='checkbox'
							checked={form.enabled}
							onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
						/>
						Активен
					</label>
					{errors.enabled && (
						<p role='alert' className='text-sm text-danger'>
							{errors.enabled}
						</p>
					)}
					<fieldset className='flex flex-col gap-2'>
						<legend className='mb-2 font-medium'>Область действия</legend>
						<label className='flex min-h-11 items-center gap-2'>
							<input
								type='radio'
								name='target'
								checked={form.target === 'all'}
								onChange={() => setForm({ ...form, target: 'all' })}
							/>
							Все блюда
						</label>
						<label className='flex min-h-11 items-center gap-2'>
							<input
								type='radio'
								name='target'
								checked={form.target === 'selected'}
								onChange={() => setForm({ ...form, target: 'selected' })}
							/>
							Выбранные блюда и категории
						</label>
						{errors.target && (
							<p role='alert' className='text-sm text-danger'>
								{errors.target}
							</p>
						)}
					</fieldset>
					<p className='text-sm text-muted'>
						Состав категорий меняется вместе с меню. Подходят блюда из выбранных категорий и отдельно выбранные блюда;
						скидка не удваивается.
					</p>
					{form.target === 'selected' && (
						<>
							{(categories.isError || products.isError) && (
								<p role='alert' className='text-danger'>
									Не удалось загрузить меню.{' '}
									<Button
										variant='secondary'
										onPress={() => {
											void categories.refetch()
											void products.refetch()
										}}
									>
										Повторить
									</Button>
								</p>
							)}
							{(categories.isPending || products.isPending) && <p role='status'>Загрузка меню…</p>}
							<div className='grid gap-4 sm:grid-cols-2'>
								<TargetPicker
									label='Блюда'
									items={products.data ?? []}
									selected={form.productIds}
									onChange={(productIds) => setForm({ ...form, productIds })}
									error={errors.productIds}
								/>
								<TargetPicker
									label='Категории'
									items={categories.data ?? []}
									selected={form.categoryIds}
									onChange={(categoryIds) => setForm({ ...form, categoryIds })}
									error={errors.categoryIds}
								/>
							</div>
						</>
					)}
				</fieldset>
				<div className='flex flex-wrap gap-2'>
					<Button
						type='submit'
						isPending={save.isPending}
						isDisabled={form.target === 'selected' && (!categories.isSuccess || !products.isSuccess)}
					>
						Сохранить
					</Button>
					<Button variant='secondary' isDisabled={save.isPending} onPress={onClose}>
						Отмена
					</Button>
				</div>
			</Form>
		</section>
	)
}
