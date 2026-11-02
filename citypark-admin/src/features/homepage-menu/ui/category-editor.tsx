import { Button } from '@heroui/react'
import { useId, useState } from 'react'
import { imageSrc } from '../../../shared/lib/image-src'
import type { Category, HomepageCategory, Product } from '../api'
import { categoryProducts, moveItem } from '../model/selection'

type Props = {
	selection: HomepageCategory
	category: Category | undefined
	products: Product[]
	onChange: (selection: HomepageCategory) => void
	onRemove: () => void
	onMove: (direction: -1 | 1) => void
	first: boolean
	last: boolean
	errors: Record<string, string>
	index: number
}

export function CategoryEditor({
	selection,
	category,
	products,
	onChange,
	onRemove,
	onMove,
	first,
	last,
	errors,
	index,
}: Props) {
	const [search, setSearch] = useState('')
	const searchId = useId()
	const name = category?.name ?? `Недоступная категория №${selection.categoryId}`
	const candidates = categoryProducts(products, selection.categoryId, search)
	const field = `categories[${index}]`
	return (
		<section aria-label={name} className='flex min-w-0 flex-col gap-3 rounded-xl border border-border p-3'>
			<div className='flex flex-wrap items-center justify-between gap-2'>
				<h3 className='font-semibold'>{name}</h3>
				<div className='flex flex-wrap gap-1'>
					<Button
						variant='secondary'
						aria-label={`Поднять категорию: ${name}`}
						isDisabled={first}
						onPress={() => onMove(-1)}
					>
						↑
					</Button>
					<Button
						variant='secondary'
						aria-label={`Опустить категорию: ${name}`}
						isDisabled={last}
						onPress={() => onMove(1)}
					>
						↓
					</Button>
					<Button variant='secondary' onPress={onRemove}>
						Убрать категорию
					</Button>
				</div>
			</div>
			{!category && (
				<p role='alert' className='text-danger'>
					Категория скрыта или удалена. Уберите её из списка.
				</p>
			)}
			{errors[field + '.categoryId'] && (
				<p role='alert' className='text-danger'>
					{errors[field + '.categoryId']}
				</p>
			)}
			{errors[field + '.productIds'] && (
				<p role='alert' className='text-danger'>
					{errors[field + '.productIds']}
				</p>
			)}
			<ol className='flex flex-col gap-2' aria-label={`Выбранные блюда: ${name}`}>
				{selection.productIds.map((id, productIndex) => {
					const product = products.find((p) => p.id === id && p.categoryId === selection.categoryId)
					const title = product?.name ?? `Недоступное блюдо №${id}`
					return (
						<li key={id} className='flex flex-wrap items-center gap-2'>
							{product?.images[0] && (
								<img src={imageSrc(product.images[0].url)} alt='' className='size-12 rounded-lg object-cover' />
							)}
							<span className='min-w-0 flex-1'>
								{title}
								{!product && (
									<span className='block text-sm text-danger'>
										Блюдо удалено или перенесено. Уберите его из списка.
									</span>
								)}
							</span>
							<Button
								variant='secondary'
								aria-label={`Поднять блюдо: ${title}`}
								isDisabled={productIndex === 0}
								onPress={() => onChange({ ...selection, productIds: moveItem(selection.productIds, productIndex, -1) })}
							>
								↑
							</Button>
							<Button
								variant='secondary'
								aria-label={`Опустить блюдо: ${title}`}
								isDisabled={productIndex === selection.productIds.length - 1}
								onPress={() => onChange({ ...selection, productIds: moveItem(selection.productIds, productIndex, 1) })}
							>
								↓
							</Button>
							<Button
								variant='secondary'
								aria-label={`Убрать блюдо: ${title}`}
								onPress={() =>
									onChange({ ...selection, productIds: selection.productIds.filter((item) => item !== id) })
								}
							>
								×
							</Button>
						</li>
					)
				})}
			</ol>
			{selection.productIds.length === 0 && (
				<p className='text-sm text-muted'>
					Автоматически: до трёх блюд этой категории, сначала доступные блюда с фото.
				</p>
			)}
			{category && (
				<>
					<label htmlFor={searchId}>Поиск блюд: {name}</label>
					<input
						id={searchId}
						type='search'
						value={search}
						onChange={(event) => setSearch(event.target.value)}
						className='min-h-11 rounded-lg border border-border bg-field px-3'
					/>
					<div className='max-h-56 overflow-y-auto'>
						{candidates.map((product) => (
							<label key={product.id} className='flex min-h-11 items-center gap-2 py-1'>
								<input
									type='checkbox'
									checked={selection.productIds.includes(product.id)}
									disabled={selection.productIds.length >= 3 && !selection.productIds.includes(product.id)}
									onChange={(event) =>
										onChange({
											...selection,
											productIds: event.target.checked
												? [...selection.productIds, product.id]
												: selection.productIds.filter((id) => id !== product.id),
										})
									}
								/>
								{product.name}
							</label>
						))}
						{candidates.length === 0 && <p className='text-sm text-muted'>Блюда не найдены</p>}
					</div>
				</>
			)}
		</section>
	)
}
