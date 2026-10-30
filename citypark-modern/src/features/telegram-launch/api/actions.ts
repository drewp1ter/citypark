'use server'

import { customerRequest } from '@/shared/customer'
import { setMiniSession } from '@/shared/customer'

type SessionEnvelope = { token: string; expiresAt: string }

/** Only raw Telegram initData crosses into this action. Never return a token to the client. */
export async function launchTelegram(initData: string): Promise<'ok' | 'invalid'> {
	if (!initData || new TextEncoder().encode(initData).length > 16 * 1024) return 'invalid'
	try {
		const result = await customerRequest<SessionEnvelope>('/auth/customer/telegram', {
			method: 'POST',
			body: JSON.stringify({ initData }),
		})
		await setMiniSession(result.token, result.expiresAt)
		return 'ok'
	} catch {
		return 'invalid'
	}
}
