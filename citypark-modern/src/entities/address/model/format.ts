import type { AddressFields } from './types'

export function formatAddress({ street, apartment, entrance }: AddressFields): string {
	return [street, apartment && `кв. ${apartment}`, entrance && `подъезд ${entrance}`].filter(Boolean).join(', ')
}

export function addressDuplicateKey(city: string, fullAddress: string): string {
	const normalize = (value: string) =>
		value
			.normalize('NFC')
			.replace(/\p{White_Space}+/gu, ' ')
			.trim()
			.toLowerCase()
	return JSON.stringify([normalize(city), normalize(fullAddress)])
}
