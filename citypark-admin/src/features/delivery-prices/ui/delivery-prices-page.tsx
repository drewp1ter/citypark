import { Button, Label, SearchField, Skeleton, Table } from '@heroui/react'
import { useQuery } from '@tanstack/react-query'
import { useMemo, useState } from 'react'
import type { components } from '../../../shared/api/schema'
import { formatRub } from '../../../shared/lib/format'
import { PageHeader } from '../../../shared/ui/page-header'
import { deliveryPricesQuery } from '../api'
import { DeleteDialog } from './delete-dialog'
import { DeliveryPriceDialog } from './delivery-price-dialog'
import { RowActions } from './row-actions'

export type DeliveryPrice = components['schemas']['DeliveryPrice']

// `item` is undefined when adding.
type EditState = { item?: DeliveryPrice } | null

function filterAndSort(items: DeliveryPrice[], search: string): DeliveryPrice[] {
	const needle = search.trim().toLowerCase()
	return items
		.filter((i) => needle === '' || i.name.toLowerCase().includes(needle) || i.value.toLowerCase().includes(needle))
		.sort((a, b) => a.name.localeCompare(b.name, 'ru'))
}

function Name({ item }: { item: DeliveryPrice }) {
	return (
		<>
			<div>{item.name}</div>
			{item.value !== '' && <div className='text-sm text-muted'>{item.value}</div>}
		</>
	)
}

export function DeliveryPricesPage() {
	const query = useQuery(deliveryPricesQuery)
	const [search, setSearch] = useState('')
	const [editing, setEditing] = useState<EditState>(null)
	const [deleting, setDeleting] = useState<DeliveryPrice | null>(null)
	const items = useMemo(() => filterAndSort([...(query.data ?? [])], search), [query.data, search])

	const actions = (item: DeliveryPrice) => (
		<RowActions item={item} onEdit={(i) => setEditing({ item: i })} onDelete={setDeleting} />
	)

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
	} else if (query.data.length === 0) {
		content = <p className='text-muted'>Городов пока нет. Добавьте первый.</p>
	} else if (items.length === 0) {
		content = <p className='text-muted'>Ничего не найдено</p>
	} else {
		content = (
			<>
				<Table className='hidden md:block'>
					<Table.ScrollContainer>
						<Table.Content aria-label='Цены доставки'>
							<Table.Header>
								<Table.Column isRowHeader>Город</Table.Column>
								<Table.Column>Цена</Table.Column>
								<Table.Column className='w-24'>
									<span className='sr-only'>Действия</span>
								</Table.Column>
							</Table.Header>
							<Table.Body>
								{items.map((item) => (
									<Table.Row key={item.id} id={item.id}>
										<Table.Cell>
											<Name item={item} />
										</Table.Cell>
										<Table.Cell>{formatRub(item.price)}</Table.Cell>
										<Table.Cell>{actions(item)}</Table.Cell>
									</Table.Row>
								))}
							</Table.Body>
						</Table.Content>
					</Table.ScrollContainer>
				</Table>
				<ul className='flex flex-col gap-2 md:hidden'>
					{items.map((item) => (
						<li
							key={item.id}
							className='flex items-center justify-between gap-3 rounded-xl border border-border bg-surface p-3'
						>
							<div className='min-w-0'>
								<Name item={item} />
								<div className='mt-1 font-medium'>{formatRub(item.price)}</div>
							</div>
							{actions(item)}
						</li>
					))}
				</ul>
			</>
		)
	}

	return (
		<>
			<PageHeader
				title='Доставка'
				actions={
					<Button variant='primary' onPress={() => setEditing({})}>
						Добавить город
					</Button>
				}
			/>
			<SearchField name='search' value={search} onChange={setSearch} className='mb-4 max-w-sm'>
				<Label>Поиск города</Label>
				<SearchField.Group>
					<SearchField.SearchIcon />
					<SearchField.Input />
					<SearchField.ClearButton />
				</SearchField.Group>
			</SearchField>
			{content}
			{editing !== null && (
				<DeliveryPriceDialog key={editing.item?.id ?? 'new'} item={editing.item} onClose={() => setEditing(null)} />
			)}
			<DeleteDialog item={deleting} onClose={() => setDeleting(null)} />
		</>
	)
}
