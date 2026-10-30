'use client'

import { useActionState } from 'react'
import { AppLink, Button, TextField, cardClass } from '@/shared/ui'
import { registerAction, resendAction, signInAction } from '../api/actions'
import { initialAuthState } from '../model/schema'

type Mode = 'sign-in' | 'register' | 'resend'
const actions = { 'sign-in': signInAction, register: registerAction, resend: resendAction }

export function AuthForm({ mode, returnTo }: { mode: Mode; returnTo?: string }) {
	const [state, action, pending] = useActionState(actions[mode], initialAuthState)
	const fields = state.fields ?? {}
	const values = state.values ?? {}
	return (
		<form action={action} className={cardClass('surface', 'w-full gap-5 rounded-lg p-5 md:p-8')}>
			{state.message && <p role={state.status === 'error' ? 'alert' : 'status'} className={state.status === 'error' ? 'text-danger' : 'text-text'}>{state.message}</p>}
			{mode === 'register' && <TextField key={`${mode}-${values.name ?? ''}`} label='Имя' name='name' autoComplete='name' required defaultValue={values.name} error={fields.name} />}
			<TextField key={`${mode}-${values.email ?? ''}`} label='Email' name='email' type='email' autoComplete='email' required defaultValue={values.email} error={fields.email} />
			{mode === 'register' && <TextField key={`${mode}-${values.phone ?? ''}`} label='Телефон (необязательно)' name='phone' type='tel' autoComplete='tel' defaultValue={values.phone} error={fields.phone} />}
			{mode !== 'resend' && <TextField label='Пароль' name='password' type='password' autoComplete={mode === 'register' ? 'new-password' : 'current-password'} required error={fields.password} />}
			{mode === 'sign-in' && <input type='hidden' name='returnTo' value={returnTo ?? ''} />}
			<Button type='submit' disabled={pending} size='xl'>{mode === 'register' ? 'Зарегистрироваться' : mode === 'resend' ? 'Отправить письмо повторно' : 'Войти'}</Button>
			{mode === 'sign-in' && <AppLink href='/account/register' className='min-h-11 self-start py-2 font-semibold underline'>Регистрация</AppLink>}
			{mode === 'register' && <AppLink href='/account/sign-in' className='min-h-11 self-start py-2 font-semibold underline'>Уже есть аккаунт? Войти</AppLink>}
			<AppLink href='/menu' className='min-h-11 self-start py-2 text-text-muted underline'>Продолжить без входа</AppLink>
		</form>
	)
}
