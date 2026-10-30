import { Button } from '@/shared/ui'
import { startGoogleSignIn } from '../api/google-actions'

export function GoogleButton() {
	return <form action={startGoogleSignIn}><Button type='submit' variant='outline' size='xl'>Войти через Google</Button></form>
}
