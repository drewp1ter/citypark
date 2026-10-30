import 'server-only'
import { CustomerApiError, customerRequest, getMiniSessionToken, type CustomerProfile } from '@/shared/customer'

export async function getMiniProfile(): Promise<CustomerProfile | undefined> {
	const token = await getMiniSessionToken()
	if (!token) return undefined
	try {
		return await customerRequest<CustomerProfile>('/customer/me', { headers: { Authorization: `Bearer ${token}` } })
	} catch (error) {
		if (error instanceof CustomerApiError && error.status === 401) return undefined
		throw error
	}
}
