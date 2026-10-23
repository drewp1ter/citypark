import Image from 'next/image'
import type { NewsItem } from '@/entities/article'
import { formatDate } from '@/shared/lib'
import { AppLink, cardClass, Section } from '@/shared/ui'

type NewsSectionProps = {
	title: string
	/** The latest news and two more, shown first. */
	items: readonly [NewsItem, NewsItem, NewsItem]
}

function NewsCard({ item }: { item: NewsItem }) {
	return (
		<AppLink href={`/news/${item.slug}`} className={cardClass('surface', 'rounded-lg p-2.5')}>
			<div className='relative h-[180px] overflow-hidden rounded-sm'>
				<Image
					src={item.photo.src}
					alt={item.photo.alt}
					fill
					sizes='(min-width: 1024px) 400px, 50vw'
					className='object-cover'
				/>
			</div>
			<span className='flex flex-col gap-1.5 px-3 pt-4 pb-2.5'>
				<span className='text-text-muted text-sm'>{formatDate(item.date)}</span>
				<span className='text-[21px] leading-tight font-bold'>{item.shortTitle ?? item.title}</span>
			</span>
		</AppLink>
	)
}

export function NewsSection({ title, items }: NewsSectionProps) {
	const [latest, , third] = items

	return (
		<Section
			title={title}
			action={
				<AppLink href='/news' className='tap-area hidden font-semibold md:inline'>
					Все новости ↗
				</AppLink>
			}
		>
			{/* Mobile: two cards in a rail, as in the design; the second slot shows news */}
			<div className='-mx-3 flex gap-2 overflow-x-auto px-3 pb-1 md:hidden'>
				{[latest, third].map((item) => (
					<AppLink
						key={item.slug}
						href={`/news/${item.slug}`}
						className={cardClass('surface', 'w-60 shrink-0 rounded-md p-2')}
					>
						<div className='relative h-[130px] overflow-hidden rounded-sm'>
							<Image src={item.photo.src} alt='' fill sizes='240px' className='object-cover' />
						</div>
						<span className='px-1.5 pt-2.5 pb-1 font-bold'>{item.title}</span>
					</AppLink>
				))}
			</div>

			{/* From 768 px: three news cards */}
			<div className='hidden gap-3 md:grid md:grid-cols-2 lg:grid-cols-3'>
				{items.map((item) => (
					<NewsCard key={item.slug} item={item} />
				))}
			</div>
		</Section>
	)
}
