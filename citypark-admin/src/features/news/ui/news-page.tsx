import { Button, Skeleton } from '@heroui/react'
import { useQuery } from '@tanstack/react-query'
import { getRouteApi, Link, useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { formatDate } from '../../../shared/lib/format'
import { imageSrc, siteUrl } from '../../../shared/lib/image-src'
import { PageHeader } from '../../../shared/ui/page-header'
import { PagePagination } from '../../../shared/ui/page-pagination'
import { NEWS_PAGE_SIZE, newsListQuery } from '../api'
import { DeleteNewsDialog, type NewsToDelete } from './delete-news-dialog'

const route = getRouteApi('/_app/content/news')
const actionClass =
	'inline-flex h-10 items-center rounded-lg bg-default px-4 text-sm font-medium hover:bg-default-hover focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus'

export function NewsPage() {
	const { page = 1 } = route.useSearch()
	const navigate = useNavigate({ from: '/content/news' })
	const query = useQuery(newsListQuery(page))
	const [toDelete, setToDelete] = useState<NewsToDelete | null>(null)

	let content
	if (query.isPending) {
		content = (
			<div className='flex flex-col gap-2' aria-busy='true' aria-label='Загрузка'>
				{[0, 1, 2].map((n) => (
					<Skeleton key={n} className='h-24 w-full rounded-xl' />
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
	} else if (query.data.news.length === 0) {
		content = <p className='text-muted'>Новостей пока нет</p>
	} else {
		content = (
			<ul className={`flex flex-col gap-3 ${query.isPlaceholderData ? 'opacity-60 transition-opacity' : ''}`}>
				{query.data.news.map((item) => (
					<li
						key={item.id}
						className='flex flex-col gap-3 rounded-xl border border-border bg-surface p-3 sm:flex-row sm:items-center'
					>
						{item.photo.url === '' ? (
							<div className='flex aspect-video w-full items-center justify-center rounded-lg bg-default text-sm text-muted sm:w-40 sm:shrink-0'>
								Без фото
							</div>
						) : (
							<img
								src={imageSrc(item.photo.url)}
								alt={item.photo.alt}
								loading='lazy'
								className='aspect-video w-full rounded-lg object-cover sm:w-40 sm:shrink-0'
								style={{ objectPosition: item.photo.position === '' ? undefined : item.photo.position }}
							/>
						)}
						<div className='flex min-w-0 flex-1 flex-col gap-1'>
							<h2 className='text-lg font-semibold'>{item.title}</h2>
							<p className='text-sm text-muted'>{formatDate(item.date)}</p>
							<a
								href={`${siteUrl()}/news/${item.slug}`}
								target='_blank'
								rel='noreferrer'
								className='w-fit text-sm text-accent-soft-foreground hover:underline'
							>
								На сайте ↗
							</a>
						</div>
						<div className='flex flex-wrap gap-2'>
							<Link
								to='/content/news/$slug'
								params={{ slug: item.slug }}
								aria-label={`Изменить: ${item.title}`}
								className={actionClass}
							>
								Изменить
							</Link>
							<Button
								variant='danger-soft'
								aria-label={`Удалить: ${item.title}`}
								onPress={() => setToDelete({ id: item.id, title: item.title })}
							>
								Удалить
							</Button>
						</div>
					</li>
				))}
			</ul>
		)
	}

	const count = query.data?.count
	return (
		<>
			<PageHeader
				title='Новости'
				actions={
					<Link to='/content/news/new' className={actionClass}>
						Добавить новость
					</Link>
				}
			/>
			{content}
			{count !== undefined && count > 0 && (
				<div className='mt-4 flex flex-wrap items-center justify-between gap-3'>
					<p className='text-sm text-muted'>Найдено: {count}</p>
					<PagePagination
						page={page}
						pageCount={Math.ceil(count / NEWS_PAGE_SIZE)}
						onChange={(next) => void navigate({ search: { page: next === 1 ? undefined : next } })}
					/>
				</div>
			)}
			<DeleteNewsDialog item={toDelete} onClose={() => setToDelete(null)} />
		</>
	)
}
