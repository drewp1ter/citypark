import { Button } from '@/shared/ui'

export function NotFoundPage() {
	return (
		<main className='max-w-page mx-auto flex w-full flex-col items-start gap-4 px-3 py-16 md:px-6 md:py-28'>
			<p className='heading text-[96px] md:text-[160px]' aria-hidden='true'>
				404
			</p>
			<h1 className='text-2xl font-bold md:text-3xl'>Такой страницы нет</h1>
			<p className='text-text-muted max-w-md'>Возможно, она переехала, или в ссылке опечатка. Начнём с главной?</p>
			<div className='mt-2 flex flex-wrap gap-2'>
				<Button href='/' size='xl'>
					На главную
				</Button>
				<Button href='/menu' size='xl' variant='outline'>
					Меню доставки
				</Button>
			</div>
		</main>
	)
}
