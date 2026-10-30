'use client'

import { use, useActionState } from 'react'
import { resetAction } from '../api/actions'
import { initialAuthState } from '../model/schema'
import { AppLink, Button, TextField, cardClass } from '@/shared/ui'

export function ResetPage({ searchParams }: { searchParams: Promise<{ token?: string }> }) {
	const token = use(searchParams).token ?? ''
	const [state, action, pending] = useActionState(resetAction, initialAuthState)
	return <main className='max-w-page mx-auto w-full px-3 py-10 md:px-6 md:py-16'><div className='mx-auto flex max-w-lg flex-col gap-6'>
		<h1 className='heading text-4xl'>Новый пароль</h1>
		<form action={action} className={cardClass('surface', 'gap-5 rounded-lg p-5 md:p-8')}>
			{state.message && <p role={state.status === 'error' ? 'alert' : 'status'}>{state.message}</p>}
			{!token && <p role='alert'>Ссылка недействительна или устарела</p>}
			<input type='hidden' name='token' value={token} />
			{state.status !== 'success' && <><TextField label='Новый пароль' name='password' type='password' autoComplete='new-password' required error={state.fields?.password} /><Button type='submit' size='xl' disabled={pending || !token}>Сменить пароль</Button></>}
			{state.status === 'success' && <Button href='/account/sign-in'>Войти</Button>}
			<AppLink href='/account/recovery' className='min-h-11 self-start py-2 underline'>Запросить новую ссылку</AppLink>
		</form>
	</div></main>
}
