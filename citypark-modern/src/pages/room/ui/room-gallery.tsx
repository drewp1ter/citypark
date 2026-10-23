'use client'

import Image from 'next/image'
import { useRef, useState } from 'react'
import type { Photo } from '@/shared/lib'

const lightboxButton =
	'bg-surface text-text border-border tap-area min-h-11 rounded-pill border px-4 text-sm font-semibold hover:underline'

/**
 * Thumbnails of the room's extra photos and a lightbox over them. The lightbox is a native <dialog>, so focus is
 * trapped and Esc closes it. `main` is item 0 of the lightbox but has no thumbnail: the page shows it already.
 */
export function RoomGallery({ main, photos }: { main: Photo; photos: readonly Photo[] }) {
	const all = [main, ...photos]
	const dialogRef = useRef<HTMLDialogElement>(null)
	const opener = useRef<HTMLButtonElement | null>(null)
	const [index, setIndex] = useState(0)
	const photo = all[index]

	const open = (at: number, button: HTMLButtonElement) => {
		opener.current = button
		setIndex(at)
		dialogRef.current?.showModal()
	}
	const step = (delta: number) => setIndex((current) => (current + delta + all.length) % all.length)

	return (
		<>
			<ul className='grid grid-cols-2 gap-2 md:grid-cols-3 md:gap-3'>
				{photos.map((item, at) => (
					<li key={item.src}>
						<button
							type='button'
							onClick={(event) => open(at + 1, event.currentTarget)}
							aria-label={item.alt}
							className='relative block aspect-[3/2] w-full cursor-pointer overflow-hidden rounded-md'
						>
							<Image
								src={item.src}
								alt=''
								fill
								sizes='(min-width: 1280px) 400px, (min-width: 768px) 33vw, 50vw'
								className='object-cover'
							/>
						</button>
					</li>
				))}
			</ul>

			<dialog
				ref={dialogRef}
				aria-label='Фотографии номера'
				onClose={() => opener.current?.focus()}
				onKeyDown={(event) => {
					if (event.key === 'ArrowLeft') step(-1)
					if (event.key === 'ArrowRight') step(1)
				}}
				onClick={(event) => {
					if (event.target === event.currentTarget) event.currentTarget.close()
				}}
				className='bg-bg text-text m-auto w-[min(96vw,1100px)] rounded-lg p-3 backdrop:bg-black/70 md:p-4'
			>
				<div className='relative aspect-[3/2] w-full overflow-hidden rounded-md'>
					{/* Remounted per photo so a slow image never shows the previous one. */}
					<Image
						key={photo.src}
						src={photo.src}
						alt={photo.alt}
						fill
						sizes='(min-width: 1100px) 1068px, 96vw'
						className='object-contain'
					/>
				</div>
				<div className='mt-3 flex flex-wrap items-center justify-between gap-2'>
					<p aria-live='polite' className='text-text-muted text-sm'>
						{index + 1} из {all.length}
					</p>
					<div className='flex gap-2'>
						<button type='button' onClick={() => step(-1)} className={lightboxButton}>
							Назад
						</button>
						<button type='button' onClick={() => step(1)} className={lightboxButton}>
							Вперёд
						</button>
						<button type='button' onClick={() => dialogRef.current?.close()} className={lightboxButton}>
							Закрыть
						</button>
					</div>
				</div>
			</dialog>
		</>
	)
}
