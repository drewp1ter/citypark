import { Button, Skeleton } from '@heroui/react'
import { useQuery } from '@tanstack/react-query'
import { getRouteApi, Outlet, useNavigate } from '@tanstack/react-router'
import { PageHeader } from '../../../shared/ui/page-header'
import { ordersQuery } from '../api'
import { PAGE_SIZE, hasFilters, type OrderSearch } from '../model/search'
import { OrderFilters } from './order-filters'
import { OrdersPagination } from './orders-pagination'
import { OrdersTable } from './orders-table'

const route = getRouteApi('/_app/orders')

export function OrdersPage() {
	const search = route.useSearch()
	const navigate = useNavigate({ from: '/orders' })
	const query = useQuery(ordersQuery(search))

	function setSearch(patch: Partial<OrderSearch>) {
		void navigate({ search: (prev) => ({ ...prev, ...patch }) })
	}
	const changeFilters = (patch: Partial<OrderSearch>) => setSearch({ ...patch, page: undefined })
	const reset = () => void navigate({ search: {} })

	let content
	if (query.isPending) {
		content = (
			<div className='flex flex-col gap-2' aria-busy='true' aria-label='Загрузка'>
				{[0, 1, 2, 3].map((n) => (
					<Skeleton key={n} className='h-14 w-full rounded-lg' />
				))}
			</div>
		)
	} else if (query.isError) {
		content = (
			<div role='alert' className='flex flex-col items-start gap-3'>
				<p className='text-danger'>{query.error.message}</p>
				<Button variant='secondary' onPress={() => void query.refetch()}>
					Повторить
				</Button>
			</div>
		)
	} else if (query.data.orders.length === 0) {
		content = (
			<div className='flex flex-col items-start gap-3'>
				<p className='text-muted'>Заказов не найдено</p>
				{hasFilters(search) && (
					<Button variant='secondary' onPress={reset}>
						Сбросить фильтры
					</Button>
				)}
			</div>
		)
	} else {
		content = (
			<div className={query.isPlaceholderData ? 'opacity-60 transition-opacity' : undefined}>
				<OrdersTable orders={query.data.orders} search={search} />
			</div>
		)
	}

	const count = query.data?.count
	return (
		<>
			<PageHeader title='Заказы' />
			<OrderFilters search={search} onChange={changeFilters} onReset={reset} />
			{content}
			{count !== undefined && count > 0 && (
				<div className='mt-4 flex flex-wrap items-center justify-between gap-3'>
					<p className='text-sm text-muted'>Найдено: {count}</p>
					<OrdersPagination
						page={search.page ?? 1}
						pageCount={Math.ceil(count / PAGE_SIZE)}
						onChange={(page) => setSearch({ page: page === 1 ? undefined : page })}
					/>
				</div>
			)}
			<Outlet />
		</>
	)
}
