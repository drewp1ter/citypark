import { Button, Skeleton } from '@heroui/react'
import { useQuery } from '@tanstack/react-query'
import { formatDateTime } from '../../../shared/lib/format'
import { PageHeader } from '../../../shared/ui/page-header'
import { freeRoomsQuery } from '../api'
import { isStale, ROOM_TYPES, TOTAL_ROOMS } from '../model/room-types'
import { RoomCard } from './room-card'

export function FreeRoomsPage() {
	const query = useQuery(freeRoomsQuery)

	let content
	if (query.isPending) {
		content = (
			<div className='flex flex-col gap-3' aria-busy='true' aria-label='Загрузка'>
				<Skeleton className='h-12 w-64 rounded-lg' />
				<Skeleton className='h-24 w-full rounded-lg' />
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
	} else if (query.data === null) {
		content = <p className='text-muted'>Данных пока нет: гостиница ещё не присылала количество свободных номеров.</p>
	} else {
		const { rooms, free, updatedAt } = query.data
		content = (
			<div className='flex flex-col gap-4'>
				<p className='text-4xl font-semibold'>
					Свободно: {free} из {TOTAL_ROOMS}
				</p>
				{isStale(updatedAt, query.dataUpdatedAt) && (
					<p role='alert' className='text-warning'>
						Данные давно не обновлялись — проверьте отправку из гостиницы
					</p>
				)}
				<ul className='grid gap-3 sm:grid-cols-2 lg:grid-cols-3'>
					{ROOM_TYPES.map((type) => (
						<RoomCard key={type.key} type={type} free={rooms[type.key]} />
					))}
				</ul>
				<p className='text-sm text-muted'>Обновлено: {formatDateTime(updatedAt)}</p>
			</div>
		)
	}

	return (
		<>
			<PageHeader title='Свободные номера' />
			{content}
		</>
	)
}
