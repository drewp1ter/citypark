import { Button } from '@heroui/react'
import { useQuery } from '@tanstack/react-query'
import { useBlocker } from '@tanstack/react-router'
import { useState } from 'react'
import { ApiError } from '../../../shared/api'
import { toastError } from '../../../shared/ui/toast'
import {
	homepageCategoriesQuery,
	homepageMenuQuery,
	homepageProductsQuery,
	useSaveHomepageMenu,
	type HomepageMenu,
} from '../api'
import { moveItem } from '../model/selection'
import { CategoryEditor } from './category-editor'

export function HomepageMenuEditor() {
	const configuration = useQuery(homepageMenuQuery)
	const categories = useQuery(homepageCategoriesQuery)
	const products = useQuery(homepageProductsQuery)
	const save = useSaveHomepageMenu()
	const [draft, setDraft] = useState<HomepageMenu | null>(null)
	const [errors, setErrors] = useState<Record<string, string>>({})
	const value = draft ?? configuration.data
	const dirty = draft !== null && JSON.stringify(draft) !== JSON.stringify(configuration.data)
	useBlocker({
		shouldBlockFn: () => !window.confirm('Есть несохранённые изменения. Уйти без сохранения?'),
		enableBeforeUnload: () => dirty,
		disabled: !dirty,
	})
	const queries = [configuration, categories, products]
	const failed = queries.find((query) => query.isError)
	if (failed)
		return (
			<section aria-label='Блюда на главной' className='my-6 flex flex-col items-start gap-3' role='alert'>
				<h2 className='text-xl font-semibold'>Блюда на главной</h2>
				<p>{failed.error?.message}</p>
				<Button
					variant='secondary'
					onPress={() => {
						void Promise.all(queries.map((query) => query.refetch()))
					}}
				>
					Повторить загрузку блюд
				</Button>
			</section>
		)
	if (!value || !categories.data || !products.data) return <p role='status'>Загрузка блюд на главной…</p>
	return (
		<section
			aria-label='Блюда на главной'
			className='my-6 flex w-full max-w-3xl flex-col gap-4 rounded-xl border border-border bg-surface p-4'
		>
			<h2 className='text-xl font-semibold'>Блюда на главной</h2>
			<p className='text-sm text-muted'>
				Выберите категории и до трёх блюд для каждой. Первая категория открывается по умолчанию. Порядок можно менять
				стрелками.
			</p>
			{dirty && (
				<p role='status' className='text-sm text-muted'>
					Есть несохранённые изменения
				</p>
			)}
			{value.categories.length === 0 && (
				<p className='text-sm text-muted'>Автоматически: первые четыре видимые категории меню.</p>
			)}
			<fieldset disabled={save.isPending} className='flex min-w-0 flex-col gap-4'>
				<legend className='sr-only'>Настройки блюд на главной</legend>
				{value.categories.map((selection, index) => (
					<CategoryEditor
						key={selection.categoryId}
						selection={selection}
						category={categories.data.find((category) => category.id === selection.categoryId)}
						products={products.data}
						index={index}
						errors={errors}
						first={index === 0}
						last={index === value.categories.length - 1}
						onChange={(next) =>
							setDraft({ categories: value.categories.map((item, i) => (i === index ? next : item)) })
						}
						onRemove={() => setDraft({ categories: value.categories.filter((_, i) => i !== index) })}
						onMove={(direction) => setDraft({ categories: moveItem(value.categories, index, direction) })}
					/>
				))}
				<label htmlFor='homepage-add-category'>Добавить категорию</label>
				<select
					id='homepage-add-category'
					value=''
					className='min-h-11 w-full rounded-lg border border-border bg-field px-3'
					onChange={(event) => {
						if (event.target.value)
							setDraft({
								categories: [...value.categories, { categoryId: Number(event.target.value), productIds: [] }],
							})
					}}
				>
					<option value=''>Выберите категорию</option>
					{categories.data
						.filter((category) => !value.categories.some((item) => item.categoryId === category.id))
						.map((category) => (
							<option key={category.id} value={category.id}>
								{category.name}
							</option>
						))}
				</select>
				{errors.categories && (
					<p role='alert' className='text-danger'>
						{errors.categories}
					</p>
				)}
				<div className='flex flex-wrap gap-2'>
					<Button
						isDisabled={!dirty}
						isPending={save.isPending}
						onPress={() => {
							setErrors({})
							save.mutate(value, {
								onSuccess: () => setDraft(null),
								onError: (error) => {
									if (error instanceof ApiError) setErrors(error.fields ?? {})
									toastError(error)
								},
							})
						}}
					>
						Сохранить блюда
					</Button>
					<Button variant='secondary' onPress={() => setDraft({ categories: [] })}>
						Вернуть автоматический выбор
					</Button>
				</div>
			</fieldset>
		</section>
	)
}
