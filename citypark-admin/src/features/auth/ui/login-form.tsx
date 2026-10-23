import { Button, FieldError, Form, Input, Label, Spinner, TextField } from '@heroui/react'
import { useRouter } from '@tanstack/react-router'
import { useState, type FormEvent } from 'react'
import { ApiError } from '../../../shared/api'
import { useSignIn } from '../api'
import { safeNextPath } from '../model/next-path'

export function LoginForm({ next }: { next: string | undefined }) {
	const router = useRouter()
	const signIn = useSignIn()
	const [email, setEmail] = useState('')
	const [password, setPassword] = useState('')

	const error = signIn.error instanceof ApiError ? signIn.error : null
	const fields = error?.status === 400 ? error.fields : {}
	const formMessage = error && Object.keys(fields).length === 0 ? error.message : null

	function submit(event: FormEvent<HTMLFormElement>) {
		event.preventDefault()
		signIn.mutate({ email, password }, { onSuccess: () => router.history.push(safeNextPath(next)) })
	}

	return (
		<Form onSubmit={submit} className='flex w-full flex-col gap-4'>
			<TextField
				isRequired
				name='email'
				type='email'
				autoComplete='username'
				value={email}
				onChange={setEmail}
				isInvalid={fields.email !== undefined}
			>
				<Label>E-mail</Label>
				<Input />
				<FieldError>{fields.email}</FieldError>
			</TextField>
			<TextField
				isRequired
				name='password'
				type='password'
				autoComplete='current-password'
				value={password}
				onChange={setPassword}
				isInvalid={fields.password !== undefined}
			>
				<Label>Пароль</Label>
				<Input />
				<FieldError>{fields.password}</FieldError>
			</TextField>
			{formMessage !== null && (
				<p role='alert' className='text-sm text-danger'>
					{formMessage}
				</p>
			)}
			<Button type='submit' variant='primary' isPending={signIn.isPending}>
				{({ isPending }) => (
					<>
						{isPending && <Spinner color='current' size='sm' />}
						Войти
					</>
				)}
			</Button>
		</Form>
	)
}
