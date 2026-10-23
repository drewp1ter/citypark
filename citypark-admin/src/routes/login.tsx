import { createFileRoute, redirect } from '@tanstack/react-router'
import { LoginForm } from '../features/auth/ui/login-form'
import { getSession } from '../shared/api'
import { ThemeToggle } from '../shared/ui/theme-toggle'

export const Route = createFileRoute('/login')({
	validateSearch: (search: Record<string, unknown>): { next?: string } =>
		typeof search.next === 'string' ? { next: search.next } : {},
	beforeLoad: () => {
		if (getSession()) throw redirect({ to: '/orders' })
	},
	component: LoginPage,
})

function LoginPage() {
	const { next } = Route.useSearch()
	return (
		<main className='flex min-h-screen items-center justify-center bg-background p-4 text-foreground'>
			<div className='flex w-full max-w-sm flex-col gap-6 rounded-2xl border border-border bg-surface p-6'>
				<div className='flex items-center justify-between'>
					<h1 className='text-2xl font-semibold'>Сити Парк</h1>
					<ThemeToggle />
				</div>
				<p className='text-sm text-muted'>Вход в панель администратора</p>
				<LoginForm next={next} />
			</div>
		</main>
	)
}
