import { createFileRoute } from '@tanstack/react-router'
import { ThemeToggle } from '../shared/ui/theme-toggle'

export const Route = createFileRoute('/')({
	component: IndexPage,
})

function IndexPage() {
	return (
		<main className='flex min-h-screen items-center justify-center gap-4 bg-background p-4 text-foreground'>
			<h1 className='text-2xl font-semibold'>Сити Парк — админка</h1>
			<ThemeToggle />
		</main>
	)
}
