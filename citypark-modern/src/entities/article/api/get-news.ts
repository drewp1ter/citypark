import { blogAuthor, blogPosts, news } from '../model/news.data'
import type { BlogAuthor, BlogPost, NewsItem } from '../model/types'

// Reads go through these functions only; swap the bodies when a CMS appears.

export async function getNews(): Promise<readonly NewsItem[]> {
	return news
}

export async function getNewsItem(slug: string): Promise<NewsItem | undefined> {
	return news.find((item) => item.slug === slug)
}

export async function getBlogPosts(): Promise<readonly BlogPost[]> {
	return blogPosts
}

export async function getBlogPost(slug: string): Promise<BlogPost | undefined> {
	return blogPosts.find((post) => post.slug === slug)
}

export async function getBlogAuthor(): Promise<BlogAuthor> {
	return blogAuthor
}
