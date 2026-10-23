import { Button, Skeleton } from '@heroui/react'
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { roomsQuery } from '../api'
import { RoomForm } from './room-form'

export function RoomEditPage({ roomId }: { roomId: string }) {
	const query = useQuery(roomsQuery)
	const room = query.data?.find((r) => r.id === roomId)

	let content
	if (query.isPending) {
		content = <Skeleton className='h-64 w-full max-w-2xl rounded-xl' aria-label='Загрузка' />
	} else if (query.isError) {
		content = (
			<div role='alert' className='flex flex-col items-start gap-3'>
				<p className='text-danger'>{query.error.message}</p>
				<Button variant='secondary' onPress={() => void query.refetch()}>
					Повторить
				</Button>
			</div>
		)
	} else if (room === undefined) {
		content = <p role='alert'>Такого номера нет</p>
	} else {
		content = <RoomForm key={room.id} room={room} />
	}

	return (
		<>
			<Link to='/content/rooms' className='mb-3 inline-block text-sm text-accent-soft-foreground hover:underline'>
				← Номера
			</Link>
			{content}
		</>
	)
}
