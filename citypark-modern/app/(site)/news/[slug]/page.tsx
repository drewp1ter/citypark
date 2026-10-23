import { ArticlePage, generateArticleMetadata } from '@/pages/news-article'

export async function generateMetadata({ params }: PageProps<'/news/[slug]'>) {
	return generateArticleMetadata((await params).slug)
}

export default async function Page({ params }: PageProps<'/news/[slug]'>) {
	return <ArticlePage slug={(await params).slug} />
}
