import { ArticlePage, generateArticleMetadata, generateArticleParams } from '@/pages/news-article'

// Only the slugs from generateStaticParams exist; anything else is a 404.
export const dynamicParams = false

export const generateStaticParams = generateArticleParams

export async function generateMetadata({ params }: PageProps<'/news/[slug]'>) {
	return generateArticleMetadata((await params).slug)
}

export default async function Page({ params }: PageProps<'/news/[slug]'>) {
	return <ArticlePage slug={(await params).slug} />
}
