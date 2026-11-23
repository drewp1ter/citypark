import { Button } from '@/shared/ui'
import { startGoogleSignIn, startYandexSignIn } from '../api/google-actions'

export function GoogleButton({ provider = 'google' }: { provider?: 'google' | 'yandex' }) {
	const yandex = provider === 'yandex'
	return <form action={yandex ? startYandexSignIn : startGoogleSignIn}><Button type='submit' variant='outline' size='xl'>{yandex ? 'Войти с Яндекс ID' : 'Войти через Google'}</Button></form>
}
