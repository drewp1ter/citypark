import { getNews } from '@/entities/article'
import { NewsFeed } from './news-feed'
import { SocialCta } from './social-cta'

export async function NewsPage() {
	const news = await getNews()

	return (
		<main>
			<NewsFeed items={news} />
			<SocialCta />
		</main>
	)
}
