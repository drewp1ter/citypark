import Image from 'next/image'
import { AppLink } from '@/shared/ui'
import { scanSections } from '../model/scans.data'

export function DocumentsPage() {
	return (
		<main className='max-w-page mx-auto w-full px-3 pt-8 pb-16 md:px-6 md:pt-12'>
			<h1 className='font-display text-[32px] leading-none font-medium tracking-[-0.03em] md:text-[length:var(--text-h1)]'>
				Документы
			</h1>
			{scanSections.map(section => (
				<section key={section.id} id={section.id} className='mt-8 scroll-mt-24 md:mt-12'>
					<h2 className='text-xl font-semibold md:text-2xl'>{section.title}</h2>
					{section.id === 'privacy-policy' && (
						<AppLink href='/privacy' className='tap-area mt-2 inline-block font-semibold'>
							Текст политики
						</AppLink>
					)}
					<ul className='mt-4 grid grid-cols-2 gap-3 md:grid-cols-4 md:gap-4'>
						{section.pages.map((page, i) => (
							<li key={page.src}>
								<a
									href={page.src}
									target='_blank'
									rel='noopener noreferrer'
									className='border-border block overflow-hidden rounded-lg border'
								>
									<Image
										src={page.src}
										alt={`${section.title}, страница ${i + 1}`}
										width={page.width}
										height={page.height}
										sizes='(min-width: 1280px) 300px, (min-width: 768px) 25vw, 50vw'
										className='h-auto w-full'
									/>
								</a>
							</li>
						))}
					</ul>
				</section>
			))}
		</main>
	)
}
