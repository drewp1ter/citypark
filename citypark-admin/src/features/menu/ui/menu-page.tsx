import { Button, Label, SearchField, Skeleton } from '@heroui/react'
import { useQuery } from '@tanstack/react-query'
import { getRouteApi, useNavigate } from '@tanstack/react-router'
import { useMemo } from 'react'
import { PageHeader } from '../../../shared/ui/page-header'
import { categoriesQuery, productsQuery } from '../api'
import { filterDishes, findGroup, groupMenu } from '../model/group'
import type { MenuSearch } from '../model/search'
import { CategoryList } from './category-list'
import { DishRow } from './dish-row'
import { SyncButton } from './sync-button'

const route = getRouteApi('/_app/menu')

export function MenuPage() {
	const search = route.useSearch()
	const navigate = useNavigate({ from: '/menu' })
	const categories = useQuery(categoriesQuery)
	const products = useQuery(productsQuery)

	const groups = useMemo(
		() => (categories.data && products.data ? groupMenu(categories.data, products.data) : []),
		[categories.data, products.data]
	)

	function setSearch(patch: Partial<MenuSearch>) {
		void navigate({ search: (prev) => ({ ...prev, ...patch }), replace: true })
	}

	let content
	if (categories.isPending || products.isPending) {
		content = (
			<div className='flex flex-col gap-2' aria-busy='true' aria-label='Загрузка'>
				{[0, 1, 2, 3].map((n) => (
					<Skeleton key={n} className='h-20 w-full rounded-xl' />
				))}
			</div>
		)
	} else if (categories.isError || products.isError) {
		const error = categories.error ?? products.error
		content = (
			<div role='alert' className='flex flex-col items-start gap-3'>
				<p className='text-danger'>{error?.message}</p>
				<Button
					variant='secondary'
					onPress={() => {
						void categories.refetch()
						void products.refetch()
					}}
				>
					Повторить
				</Button>
			</div>
		)
	} else if (groups.length === 0) {
		content = <p className='text-muted'>Меню пока пусто. Нажмите «Обновить меню из iiko».</p>
	} else {
		const selected = findGroup(groups, search.category)
		const pool = selected !== undefined ? selected.dishes : groups.flatMap((g) => g.dishes)
		const dishes = filterDishes(pool, search.q)
		content = (
			<div className='flex flex-col gap-4 lg:flex-row lg:items-start lg:gap-6'>
				<div className='lg:w-64 lg:shrink-0'>
					<CategoryList
						groups={groups}
						total={products.data.length}
						selected={selected === undefined ? undefined : search.category}
						onSelect={(category) => setSearch({ category })}
					/>
				</div>
				<div className='flex min-w-0 flex-1 flex-col gap-3'>
					<SearchField
						name='q'
						value={search.q ?? ''}
						onChange={(value) => setSearch({ q: value.trim() === '' ? undefined : value })}
						className='w-full sm:w-72'
					>
						<Label>Поиск блюда</Label>
						<SearchField.Group>
							<SearchField.SearchIcon />
							<SearchField.Input />
							<SearchField.ClearButton />
						</SearchField.Group>
					</SearchField>
					{dishes.length === 0 ? (
						<p className='text-muted'>Блюд не найдено</p>
					) : (
						<ul className='flex flex-col gap-2'>
							{dishes.map((dish) => (
								<DishRow key={dish.id} dish={dish} />
							))}
						</ul>
					)}
				</div>
			</div>
		)
	}

	return (
		<>
			<PageHeader title='Меню' actions={<SyncButton />} />
			<p className='mb-4 text-sm text-muted'>
				Названия, цены, состав и фото меняются в iiko, затем нажмите «Обновить меню из iiko».
			</p>
			{content}
		</>
	)
}
