import { Button, Skeleton } from '@heroui/react'
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { ApiError } from '../../../shared/api'
import { newsItemQuery } from '../api'
import { toForm } from '../model/form'
import { NewsForm } from './news-form'

function BackLink() {
	return (
		<Link to='/content/news' className='mb-3 inline-block text-sm text-accent-soft-foreground hover:underline'>
			← Новости
		</Link>
	)
}

export function NewsEditPage({ slug }: { slug: string }) {
	const query = useQuery(newsItemQuery(slug))

	let content
	if (query.isPending) {
		content = <Skeleton className='h-64 w-full max-w-2xl rounded-xl' aria-label='Загрузка' />
	} else if (query.isError) {
		const missing = query.error instanceof ApiError && query.error.status === 404
		content = missing ? (
			<p role='alert'>Такой новости нет</p>
		) : (
			<div role='alert' className='flex flex-col items-start gap-3'>
				<p className='text-danger'>{query.error.message}</p>
				<Button variant='secondary' onPress={() => void query.refetch()}>
					Повторить
				</Button>
			</div>
		)
	} else {
		content = (
			<>
				<h1 className='mb-6 text-2xl font-semibold'>{query.data.title}</h1>
				<NewsForm key={query.data.id} id={query.data.id} initial={toForm(query.data)} />
			</>
		)
	}

	return (
		<>
			<BackLink />
			{content}
		</>
	)
}
