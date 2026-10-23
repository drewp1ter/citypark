import Image from 'next/image'
import { notFound } from 'next/navigation'
import type { Metadata } from 'next'
import { getNewsItem } from '@/entities/article'
import { formatDate, type Photo } from '@/shared/lib'
import { AppLink, Button, RichText } from '@/shared/ui'

type Article = { title: string; meta: string; photo: Photo; bodyHtml: string }

async function findArticle(slug: string): Promise<Article | undefined> {
	const item = await getNewsItem(slug)
	if (!item) return undefined
	return { title: item.title, meta: formatDate(item.date), photo: item.photo, bodyHtml: item.bodyHtml }
}

export async function generateArticleMetadata(slug: string): Promise<Metadata> {
	const article = await findArticle(slug)
	return { title: article ? `${article.title} — Сити Парк` : 'Страница не найдена' }
}

export async function ArticlePage({ slug }: { slug: string }) {
	const article = await findArticle(slug)
	if (!article) notFound()

	return (
		<main className='max-w-page mx-auto w-full px-3 pt-4 pb-6 md:px-6 md:pt-8'>
			<AppLink href='/news' className='tap-area mb-4 inline-block font-semibold md:mb-6'>
				← Все новости
			</AppLink>
			<article className='mx-auto max-w-3xl'>
				<p className='text-text-muted text-sm'>{article.meta}</p>
				<h1 className='heading mt-2 text-[32px] md:text-5xl'>{article.title}</h1>
				<div className='relative my-6 h-[220px] overflow-hidden rounded-lg md:my-8 md:h-[420px] md:rounded-xl'>
					<Image
						src={article.photo.src}
						alt={article.photo.alt}
						fill
						loading='eager'
						fetchPriority='high'
						sizes='(min-width: 768px) 768px, 100vw'
						className='object-cover'
					/>
				</div>
				{article.bodyHtml ? (
					<RichText html={article.bodyHtml} variant='article' className='text-lg leading-relaxed' />
				) : (
					<p className='text-text-muted'>Текст статьи скоро появится.</p>
				)}
				<div className='mt-8 flex flex-wrap gap-2'>
					<Button href='/menu' size='xl'>
						Меню доставки
					</Button>
					<Button href='/restaurant' size='xl' variant='outline'>
						Забронировать столик
					</Button>
				</div>
			</article>
		</main>
	)
}
