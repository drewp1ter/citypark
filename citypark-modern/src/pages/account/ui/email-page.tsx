'use client'

import { use, useActionState } from 'react'
import { confirmEmailAction } from '../api/actions'
import { initialAuthState } from '../model/schema'
import { AppLink, Button, cardClass } from '@/shared/ui'

export function EmailPage({ searchParams }: { searchParams: Promise<{ token?: string }> }) {
	const token = use(searchParams).token ?? ''
	const [state, action, pending] = useActionState(confirmEmailAction, initialAuthState)
	return <main className='max-w-page mx-auto w-full px-3 py-10 md:px-6 md:py-16'><div className='mx-auto flex max-w-lg flex-col gap-6'>
		<h1 className='heading text-4xl'>Подтверждение нового email</h1>
		<form action={action} className={cardClass('surface', 'gap-5 rounded-lg p-5 md:p-8')}>
			{state.message && <p role={state.status === 'error' ? 'alert' : 'status'}>{state.message}</p>}
			{!token && <p role='alert'>Ссылка недействительна или устарела</p>}
			<input type='hidden' name='token' value={token} />
			{state.status !== 'success' && <Button type='submit' size='xl' disabled={pending || !token}>Подтвердить email</Button>}
			<AppLink href='/account/sign-in' className='min-h-11 self-start py-2 underline'>Войти повторно</AppLink>
		</form>
	</div></main>
}
