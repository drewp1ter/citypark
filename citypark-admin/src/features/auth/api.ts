import { useMutation } from '@tanstack/react-query'
import { api, ApiError, setSession, unwrap } from '../../shared/api'

export const NOT_ADMIN_MESSAGE = 'Нет доступа: нужен аккаунт администратора'
const WRONG_CREDENTIALS_MESSAGE = 'Неверный e-mail или пароль'

type Credentials = { email: string; password: string }

/** Signs in and stores the session. Throws ApiError; a non-admin account is rejected before anything is stored. */
async function signIn({ email, password }: Credentials): Promise<void> {
	let user
	try {
		user = await unwrap(api.POST('/auth/sign-in', { body: { email, password } }))
	} catch (e) {
		if (e instanceof ApiError && e.status === 401) throw new ApiError(401, WRONG_CREDENTIALS_MESSAGE)
		throw e
	}
	if (user.role !== 'admin') throw new ApiError(403, NOT_ADMIN_MESSAGE)
	setSession({ token: user.token, email: user.email, name: user.name })
}

export function useSignIn() {
	return useMutation({ mutationFn: signIn })
}
