'use client'

import { captureAddressAutofillFailure, type AutofillReason } from '@/shared/monitoring'
import { useEffect, useRef, useState } from 'react'

type Result =
	{ status: 'matched'; town: string; street: string } | { status: 'unmatched' } | { status: 'unavailable' | 'invalid' }
type Options = {
	surface: 'web' | 'mini'
	enabled: boolean
	configured: boolean
	initialTown: string
	initialStreet: string
	onDetected: (town: string, street: string) => void
	onFailure: () => void
}

export function useCheckoutLocation({ surface, enabled, configured, initialTown, initialStreet, onDetected, onFailure }: Options) {
	const [status, setStatus] = useState('')
	const started = useRef(false)
	const takenOver = useRef(false)
	const cancelled = useRef(false)
	const active = useRef(false)

	useEffect(() => {
		if (!enabled || !configured || initialTown || initialStreet || cancelled.current) return
		if (started.current) {
			active.current = true
			return () => {
				active.current = false
			}
		}
		const fail = (message = 'Не удалось определить адрес. Проверьте населённый пункт и введите улицу и номер дома.') => {
			if (takenOver.current || cancelled.current) return
			onFailure()
			setStatus(message)
		}
		const begin = Date.now()
		const report = (reason: AutofillReason, browser_code?: number) =>
			captureAddressAutofillFailure(reason, {
				stage: 'browser',
				surface,
				elapsed_ms: Date.now() - begin,
				secure_context: window.isSecureContext,
				browser_code,
			})
		started.current = true
		if (!navigator.geolocation) {
			report('unsupported')
			queueMicrotask(() => fail())
			return
		}
		active.current = true
		queueMicrotask(() => {
			if (active.current && !takenOver.current) setStatus('Определяем ваш адрес…')
		})
		navigator.geolocation.getCurrentPosition(
			async ({ coords }) => {
				if (!active.current || takenOver.current) return
				try {
					const response = await fetch('/api/address-geolocate', {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({ lat: coords.latitude, lon: coords.longitude }),
						signal: AbortSignal.timeout(5000),
					})
					const result = (await response.json()) as Result
					if (!result || !['matched', 'unmatched', 'unavailable', 'invalid'].includes(result.status))
						throw new Error('Invalid autofill response')
					if (result.status === 'matched' && (typeof result.town !== 'string' || typeof result.street !== 'string'))
						throw new Error('Invalid autofill address')
					if (!active.current || takenOver.current) return
					if (response.ok && result.status === 'matched') {
						onDetected(result.town, result.street)
						setStatus('Адрес определён. Проверьте улицу и номер дома.')
					} else if (response.ok && result.status === 'unmatched')
						fail()
					else fail()
				} catch (error) {
					if (active.current && !takenOver.current)
						report(
							error instanceof Error && error.name === 'TimeoutError'
								? 'request_timeout'
								: error instanceof SyntaxError ||
									  (error instanceof Error && error.message.startsWith('Invalid autofill'))
									? 'invalid_response'
									: 'request_failed'
						)
					if (active.current && !takenOver.current)
						fail()
				}
			},
			(error) => {
				if (active.current && !takenOver.current)
					report(
						error.code === 1 ? 'permission_denied' : error.code === 3 ? 'location_timeout' : 'position_unavailable',
						error.code
					)
				if (active.current && !takenOver.current)
					fail()
			},
			{ timeout: 30000, maximumAge: 0, enableHighAccuracy: false }
		)
		return () => {
			active.current = false
		}
	}, [surface, enabled, configured, initialTown, initialStreet, onDetected, onFailure])

	return {
		status,
		takeOver: () => {
			takenOver.current = true
			cancelled.current = true
			active.current = false
			cancelled.current = true
			setStatus('')
		},
		pickup: () => {
			active.current = false
			setStatus('')
		},
	}
}
