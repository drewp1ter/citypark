import Image from 'next/image'
import { notFound } from 'next/navigation'
import type { Metadata } from 'next'
import { getRoom, roomTourHref, type Room } from '@/entities/room'
import { getSiteSettings } from '@/shared/api'
import { formatPrice } from '@/shared/lib'
import { AppLink, Breadcrumbs, Button, RichText } from '@/shared/ui'
import { RoomGallery } from './room-gallery'

/** `undefined` = no such room; a backend failure is logged and reported as `null`. */
async function findRoom(id: string): Promise<Room | undefined | null> {
	try {
		return await getRoom(id)
	} catch (error) {
		console.error(error)
		return null
	}
}

export async function generateRoomMetadata(id: string): Promise<Metadata> {
	const room = await findRoom(id)
	if (!room) return { title: 'Страница не найдена' }
	return { title: `${room.name} — Сити Парк`, ...(room.summary && { description: room.summary }) }
}

export async function RoomPage({ id }: { id: string }) {
	const room = await findRoom(id)
	if (room === undefined) notFound()

	if (room === null) {
		const { phones } = await getSiteSettings()
		return (
			<main className='max-w-page mx-auto flex w-full flex-col items-start gap-4 px-3 pt-4 pb-16 md:px-6 md:pt-8 md:pb-28'>
				<h1 className='heading mx-1.5 mt-2 text-[38px] md:mx-0 md:mt-0 md:text-[length:var(--text-h1)]'>Номера</h1>
				<p className='text-text-muted mx-1.5 md:mx-0 md:text-lg'>Не удалось загрузить номер. Позвоните нам:</p>
				<Button href={phones.hotel.href} size='xl' variant='outline'>
					{phones.hotel.display}
				</Button>
			</main>
		)
	}

	return (
		<main className='max-w-page mx-auto w-full px-3 pt-4 pb-16 md:px-6 md:pt-8 md:pb-28'>
			<Breadcrumbs items={[{ label: 'Номера', href: '/rooms' }, { label: room.name }]} />
			<h1 className='heading text-[32px] md:text-5xl'>{room.name}</h1>
			<p className='text-text-muted mt-2 md:text-lg'>
				{room.numbers} · {room.areaDetailed ?? room.area}
				{room.capacity && ` · ${room.capacity}`}
			</p>

			<div className='mt-6 grid gap-6 md:mt-8 md:grid-cols-2 md:gap-10'>
				<div className='relative h-[240px] overflow-hidden rounded-lg md:h-[420px] md:rounded-xl'>
					<Image
						src={room.photo.src}
						alt={room.photo.alt}
						fill
						loading='eager'
						fetchPriority='high'
						sizes='(min-width: 1280px) 600px, (min-width: 768px) 50vw, 100vw'
						style={room.photo.position ? { objectPosition: room.photo.position } : undefined}
						className='object-cover'
					/>
				</div>
				<div className='flex flex-col items-start gap-3'>
					<p>
						<span className='font-display text-[30px] font-medium'>{formatPrice(room.pricePerNight)}</span>
						<span className='text-text-muted'> / ночь</span>
					</p>
					<Button href='/rooms#search' variant='accent' className='min-h-[50px]'>
						Забронировать
					</Button>
					<a
						href={roomTourHref(room.kind)}
						target='_blank'
						rel='noopener'
						className='text-text-muted inline-flex min-h-11 items-center text-sm hover:underline'
					>
						3D-тур ↗<span className='sr-only'> (откроется в новой вкладке)</span>
					</a>
				</div>
			</div>

			{room.gallery.length > 0 && (
				<section className='mt-10 md:mt-14'>
					<h2 className='heading mb-4 text-2xl md:text-4xl'>Фото</h2>
					<RoomGallery main={room.photo} photos={room.gallery} />
				</section>
			)}

			{room.descriptionHtml && (
				<RichText
					html={room.descriptionHtml}
					variant='article'
					className='mt-10 max-w-3xl text-lg leading-relaxed md:mt-14'
				/>
			)}

			<p className='text-text-muted mt-10 max-w-3xl'>
				Правила проживания, дополнительные услуги и документы — на{' '}
				<AppLink href='/hotel#info' className='text-text tap-area font-semibold underline'>
					странице отеля
				</AppLink>
				.
			</p>
		</main>
	)
}
