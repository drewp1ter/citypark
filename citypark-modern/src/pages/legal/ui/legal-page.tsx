// Placeholder for Документы / Оферта / Конфиденциальность: the owner has not sent the texts yet.
export function LegalPage({ title }: { title: string }) {
	return (
		<main className='max-w-page mx-auto w-full px-3 pt-8 pb-16 md:px-6 md:pt-12'>
			<h1 className='font-display text-[32px] leading-none font-medium tracking-[-0.03em] md:text-[length:var(--text-h1)]'>
				{title}
			</h1>
			<p className='text-text-muted mt-6'>Раздел скоро появится.</p>
		</main>
	)
}
