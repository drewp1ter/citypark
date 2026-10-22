import Image from 'next/image'
import type { BlogAuthor, BlogPost } from '@/entities/article'
import { AppLink, cardClass } from '@/shared/ui'

type ChefBlogProps = { author: BlogAuthor; posts: readonly BlogPost[] }

/** The chef's blog. Desktop: an intro card and one card per recipe. Mobile: one dark block with a scrolling row. */
export function ChefBlog({ author, posts }: ChefBlogProps) {
	return (
		<section id='blog' className='max-w-page mx-auto w-full scroll-mt-24 px-3 pt-10 md:px-6 md:pt-[104px]'>
			{/* Mobile */}
			<div className={cardClass('inverse', 'gap-3.5 rounded-lg p-5 md:hidden')}>
				<div className='flex items-center gap-3'>
					<Image
						src={author.photo.src}
						alt={author.photo.alt}
						width={60}
						height={60}
						className='border-accent size-[60px] rounded-full border-2 object-cover object-top'
					/>
					<div>
						<h2 className='font-display text-xl font-medium'>Блог шефа</h2>
						<p className='text-inverse-muted text-sm'>{author.name}</p>
					</div>
				</div>
				<div className='-mx-5 flex gap-2 overflow-x-auto px-5'>
					{posts.map((post) => (
						<AppLink
							key={post.slug}
							href={`/news/${post.slug}`}
							className='bg-inverse-surface w-[200px] shrink-0 rounded-md p-1.5'
						>
							<div className='relative h-[120px] overflow-hidden rounded-xs'>
								{/* The title below says the same as the alt text, so the photo is decorative here. */}
								<Image src={post.photo.src} alt='' fill sizes='200px' className='object-cover' />
							</div>
							<span className='block px-1.5 pt-2.5 pb-1 font-bold'>{post.shortTitle ?? post.title}</span>
						</AppLink>
					))}
				</div>
			</div>

			{/* Desktop */}
			<div className='grid gap-3 max-md:hidden md:grid-cols-2 lg:grid-cols-4'>
				<div className={cardClass('inverse', 'justify-between gap-4 rounded-xl p-7')}>
					<Image
						src={author.photo.src}
						alt={author.photo.alt}
						width={88}
						height={88}
						className='border-accent size-[88px] rounded-full border-[3px] object-cover object-top'
					/>
					<div>
						<h2 className='font-display mb-2 text-[26px] leading-[1.1] font-medium tracking-[-0.02em]'>
							Блог шеф-повара
						</h2>
						<p className='text-inverse-muted text-[15px]'>{author.name} делится рецептами и секретами кухни.</p>
					</div>
				</div>
				{posts.map((post) => (
					<AppLink key={post.slug} href={`/news/${post.slug}`} className={cardClass('surface', 'rounded-lg p-2.5')}>
						<div className='relative h-[170px] overflow-hidden rounded-sm'>
							<Image
								src={post.photo.src}
								alt={post.photo.alt}
								fill
								sizes='(min-width: 1024px) 300px, 50vw'
								className='object-cover'
							/>
						</div>
						<span className='flex flex-col gap-1 px-2.5 pt-3.5 pb-2'>
							<span className='text-[19px] font-bold'>{post.title}</span>
							<span className='text-text-muted text-sm'>{post.subtitle}</span>
						</span>
					</AppLink>
				))}
			</div>
		</section>
	)
}
