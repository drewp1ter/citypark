import { getNews } from '@/entities/article'
import { NewsFeed } from './news-feed'
import { SocialCta } from './social-cta'

export async function NewsPage() {
	const news = await getNews().catch((error: unknown) => {
		console.error(error)
		return null
	})

	return (
		<main>
			{news?.length ? (
				<NewsFeed items={news} />
			) : (
				<div className='max-w-page mx-auto flex w-full flex-col items-start gap-4 px-3 pt-4 pb-16 md:px-6 md:pt-8'>
					<h1 className='heading mx-1.5 mt-2 text-4xl md:mx-0 md:mt-0 md:text-[length:var(--text-h1)]'>Новости</h1>
					<p className='text-text-muted mx-1.5 md:mx-0 md:text-lg'>
						{news ? 'Новостей пока нет.' : 'Не удалось загрузить новости.'}
					</p>
				</div>
			)}
			<SocialCta />
		</main>
	)
}
