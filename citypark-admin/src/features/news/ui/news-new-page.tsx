import { Link } from '@tanstack/react-router'
import { moscowToday } from '../../../shared/lib/today'
import { emptyNews } from '../model/form'
import { NewsForm } from './news-form'

export function NewsNewPage() {
	return (
		<>
			<Link to='/content/news' className='mb-3 inline-block text-sm text-accent-soft-foreground hover:underline'>
				← Новости
			</Link>
			<h1 className='mb-6 text-2xl font-semibold'>Новая новость</h1>
			<NewsForm initial={emptyNews(moscowToday())} />
		</>
	)
}
