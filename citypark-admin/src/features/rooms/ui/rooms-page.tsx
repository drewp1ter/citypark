import { Button, Skeleton } from '@heroui/react'
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { formatRub } from '../../../shared/lib/format'
import { imageSrc } from '../../../shared/lib/image-src'
import { PageHeader } from '../../../shared/ui/page-header'
import { roomsQuery } from '../api'

export function RoomsPage() {
	const query = useQuery(roomsQuery)

	let content
	if (query.isPending) {
		content = (
			<div className='grid gap-4 sm:grid-cols-2 lg:grid-cols-3' aria-busy='true' aria-label='Загрузка'>
				<Skeleton className='h-64 rounded-xl' />
				<Skeleton className='h-64 rounded-xl' />
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
	} else {
		content = (
			<ul className='grid gap-4 sm:grid-cols-2 lg:grid-cols-3'>
				{query.data.map((room) => (
					<li key={room.id} className='flex flex-col overflow-hidden rounded-xl border border-border bg-surface'>
						<img
							src={imageSrc(room.photo.url)}
							alt={room.photo.alt}
							loading='lazy'
							className='aspect-[4/3] w-full object-cover'
							style={{ objectPosition: room.photo.position === '' ? undefined : room.photo.position }}
						/>
						<div className='flex flex-1 flex-col gap-1 p-4'>
							<h2 className='text-lg font-semibold'>{room.name}</h2>
							<p className='text-sm text-muted'>{room.numbers}</p>
							<p className='mt-1'>{formatRub(room.pricePerNight)} / ночь</p>
							<Link
								to='/content/rooms/$roomId'
								params={{ roomId: room.id }}
								aria-label={`Изменить: ${room.name}`}
								className='mt-3 inline-flex h-10 w-fit items-center rounded-lg bg-default px-4 text-sm font-medium hover:bg-default-hover focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus'
							>
								Изменить
							</Link>
						</div>
					</li>
				))}
			</ul>
		)
	}

	return (
		<>
			<PageHeader title='Номера' />
			{content}
		</>
	)
}
