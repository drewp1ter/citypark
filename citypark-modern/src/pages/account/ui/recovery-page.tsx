'use client'

import { useActionState } from 'react'
import { recoveryAction } from '../api/actions'
import { initialAuthState } from '../model/schema'
import { AppLink, Button, TextField, cardClass } from '@/shared/ui'

export function RecoveryPage() {
	const [state, action, pending] = useActionState(recoveryAction, initialAuthState)
	return <main className='max-w-page mx-auto w-full px-3 py-10 md:px-6 md:py-16'><div className='mx-auto flex max-w-lg flex-col gap-6'>
		<h1 className='heading text-4xl'>Восстановление пароля</h1>
		<form action={action} className={cardClass('surface', 'gap-5 rounded-lg p-5 md:p-8')}>
			{state.message && <p role={state.status === 'error' ? 'alert' : 'status'}>{state.message}</p>}
			<TextField label='Email' name='email' type='email' autoComplete='email' required defaultValue={state.values?.email} error={state.fields?.email} />
			<Button type='submit' size='xl' disabled={pending}>Отправить ссылку</Button>
			<AppLink href='/account/sign-in' className='min-h-11 self-start py-2 underline'>Вернуться ко входу</AppLink>
		</form>
	</div></main>
}
