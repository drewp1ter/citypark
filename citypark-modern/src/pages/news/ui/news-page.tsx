import { getBlogAuthor, getBlogPosts, getNews } from '@/entities/article'
import { ChefBlog } from './chef-blog'
import { NewsFeed } from './news-feed'
import { SocialCta } from './social-cta'

export async function NewsPage() {
	const [news, posts, author] = await Promise.all([getNews(), getBlogPosts(), getBlogAuthor()])

	return (
		<main>
			<NewsFeed items={news} />
			<ChefBlog author={author} posts={posts} />
			<SocialCta />
		</main>
	)
}
