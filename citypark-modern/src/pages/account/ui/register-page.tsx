import { AuthForm } from './auth-form'

export function RegisterPage() {
	return <main className='max-w-page mx-auto w-full px-3 py-10 md:px-6 md:py-16'><div className='mx-auto flex max-w-lg flex-col gap-6'>
		<h1 className='heading text-4xl'>Регистрация</h1>
		<AuthForm mode='register' />
		<h2 className='heading text-2xl'>Повторная отправка письма</h2>
		<AuthForm mode='resend' />
	</div></main>
}
