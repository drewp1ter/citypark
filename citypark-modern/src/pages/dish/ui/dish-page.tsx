import Image from 'next/image'
import { notFound } from 'next/navigation'
import type { Metadata } from 'next'
import { getMenu } from '@/entities/dish'
import { productionOrigin, socialMetadata } from '@/shared/config'
import { Breadcrumbs, RichText } from '@/shared/ui'
import { DishControls } from './dish-controls'

async function findDish(id: string) {
	const menu = await getMenu()
	const dish = menu.dishes.find((item) => item.id === id)
	if (!dish) notFound()
	return dish
}

export async function generateDishMetadata(id: string): Promise<Metadata> {
	const dish = await findDish(id)
	const path = `/menu/${encodeURIComponent(dish.id)}`
	const title = `${dish.name} — Сити Парк`
	const description = dish.composition || `${dish.name}: фото, порции и цены. Доставка Сити Парк в Белореченске.`
	return {
		...socialMetadata(title, description, path, dish.photo.src, dish.photo.alt),
		title,
		description,
		alternates: { canonical: path },
	}
}

export async function DishPage({ id }: { id: string }) {
	const dish = await findDish(id)
	const path = `/menu/${encodeURIComponent(dish.id)}`
	return (
		<main className='max-w-page mx-auto w-full px-3 pt-4 pb-28 md:px-6 md:pt-8'>
			<Breadcrumbs
				items={[{ label: 'Меню', href: `/menu?category=${encodeURIComponent(dish.categoryId)}` }, { label: dish.name }]}
			/>
			<h1 className='heading text-[32px] break-words md:text-5xl'>{dish.name}</h1>
			<div className='mt-6 grid gap-6 md:grid-cols-2 md:gap-10'>
				<div className='relative aspect-square overflow-hidden rounded-lg'>
					<Image
						src={dish.photo.src}
						alt={dish.photo.alt}
						fill
						sizes='(min-width: 768px) 50vw, 100vw'
						loading='eager'
						className='object-cover'
					/>
				</div>
				<div className='flex min-w-0 flex-col gap-6'>
					{dish.fullDescription ? (
						<RichText html={dish.fullDescription} variant='short' className='text-text-muted break-words md:text-lg' />
					) : dish.composition ? (
						<p className='text-text-muted break-words whitespace-pre-line md:text-lg'>{dish.composition}</p>
					) : null}
					<DishControls dish={dish} url={productionOrigin + path} />
				</div>
			</div>
		</main>
	)
}
