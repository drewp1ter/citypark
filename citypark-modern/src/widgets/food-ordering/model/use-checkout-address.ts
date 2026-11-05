'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import { addressFieldsSchema, saveGuestAddress, useGuestAddresses, type AddressBookLoad } from '@/entities/address'

export type CheckoutAddress = { city: string; street: string; apartment: string; entrance: string }
export type CheckoutAddressEntry = CheckoutAddress & {
	key: string
	name: string
	isDefault: boolean
	needsReview: boolean
}
const empty: CheckoutAddress = { city: '', street: '', apartment: '', entrance: '' }

export function useCheckoutAddress(book: AddressBookLoad, towns: readonly { name: string }[]) {
	const guest = useGuestAddresses(book.context.kind === 'guest')
	const ready = book.context.kind !== 'guest' || guest.status !== 'loading'
	const entries: CheckoutAddressEntry[] =
		book.context.kind === 'guest'
			? guest.book.entries.map((entry) => ({ ...entry, key: `guest:${entry.id}`, needsReview: false }))
			: book.entries.map((entry) => ({ ...entry, key: `account:${entry.id}` }))
	const [fields, setFields] = useState<CheckoutAddress>(empty)
	const [selection, setSelection] = useState('')
	const [settled, setSettled] = useState(false)
	const [locked, setLocked] = useState(false)
	const [save, setSave] = useState(false)
	const [name, setName] = useState('')
	const [saveError, setSaveError] = useState('')
	const protectedDraft = useRef(false)
	const initialized = useRef(false)
	const eligible = (entry: CheckoutAddressEntry) =>
		!entry.needsReview && addressFieldsSchema.safeParse(entry).success && towns.some((town) => town.name === entry.city)
	const defaultEntry = entries.find((entry) => entry.isDefault && eligible(entry))
	useEffect(() => {
		if (!ready || initialized.current) return
		initialized.current = true
		queueMicrotask(() => {
			if (!protectedDraft.current && defaultEntry) {
				protectedDraft.current = true
				setLocked(true)
				setFields({
					city: defaultEntry.city,
					street: defaultEntry.street,
					apartment: defaultEntry.apartment,
					entrance: defaultEntry.entrance,
				})
				setSelection(defaultEntry.key)
			}
			setSettled(true)
		})
		// A default is considered exactly once after hydration, never after later storage events.
	}, [ready, defaultEntry])
	const selected = entries.some((entry) => entry.key === selection) ? selection : ''
	const edit = useCallback(
		(field: keyof CheckoutAddress, value: string) => {
			protectedDraft.current = true
			setLocked(true)
			setSelection('')
			setFields((current) =>
				field === 'city' ? { city: value, street: '', apartment: '', entrance: '' } : { ...current, [field]: value }
			)
		},
		[setFields, setSelection, setLocked]
	)
	const takeOver = useCallback(() => {
		protectedDraft.current = true
		setLocked(true)
		setSelection('')
	}, [setSelection, setLocked])
	const select = (key: string) => {
		const entry = entries.find((entry) => entry.key === key)
		if (key && (!entry || !eligible(entry))) return
		protectedDraft.current = true
		setLocked(true)
		setSelection(key)
		setFields(
			entry ? { city: entry.city, street: entry.street, apartment: entry.apartment, entrance: entry.entrance } : empty
		)
	}
	const detected = useCallback(
		(city: string, street: string) => {
			if (!protectedDraft.current) setFields({ city, street, apartment: '', entrance: '' })
		},
		[setFields]
	)
	const fallbackTown = towns[0]?.name || ''
	const fallback = useCallback(() => {
		if (!protectedDraft.current) setFields((current) => (current.city ? current : { ...current, city: fallbackTown }))
	}, [fallbackTown, setFields])
	async function beforeSubmit(method: string) {
		protectedDraft.current = true
		setLocked(true)
		setSaveError('')
		if (method !== 'delivery' || !save) return true
		const parsed = addressFieldsSchema.safeParse({ ...fields, name })
		if (!parsed.success) {
			setSaveError(parsed.error.issues[0].message)
			return false
		}
		if (book.context.kind === 'guest') {
			const result = await saveGuestAddress(parsed.data)
			if (!result.ok) {
				setSaveError(result.message)
				return false
			}
		}
		return true
	}
	return {
		fields,
		entries,
		selected,
		ready: ready && settled,
		locationAllowed: ready && settled && !locked,
		edit,
		takeOver,
		select,
		detected,
		fallback,
		save,
		setSave,
		name,
		setName,
		saveError,
		beforeSubmit,
	}
}
