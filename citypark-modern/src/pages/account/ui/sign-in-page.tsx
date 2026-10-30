import { AuthForm } from './auth-form'
import { logoutAction } from '../api/actions'
import { hasCustomerSessionCookie } from '@/shared/customer'
import { Button } from '@/shared/ui'

export async function SignInPage({ searchParams }: { searchParams: Promise<{ returnTo?: string; logout?: string }> }) {
	const { returnTo, logout } = await searchParams
	const signedIn = await hasCustomerSessionCookie()
	return <main className='max-w-page mx-auto w-full px-3 py-10 md:px-6 md:py-16'>
		<div className='mx-auto flex max-w-lg flex-col gap-6'>
			<h1 className='heading text-4xl'>Вход</h1>
			{logout === 'remote-failed' && <p role='alert'>Сеанс удалён с этого устройства. Не удалось завершить его на сервере. Попробуйте позже.</p>}
			{logout === '1' && <p role='status'>Вы вышли из аккаунта.</p>}
			{signedIn && <form action={logoutAction}><Button type='submit' variant='outline'>Выйти из аккаунта</Button></form>}
			<AuthForm mode='sign-in' returnTo={returnTo} />
		</div>
	</main>
}
