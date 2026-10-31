'use client'

import { useEffect, useRef, useState } from 'react'

type Result =
	{ status: 'matched'; town: string; street: string } | { status: 'unmatched' } | { status: 'unavailable' | 'invalid' }
type Options = {
	enabled: boolean
	configured: boolean
	initialTown: string
	initialStreet: string
	onDetected: (town: string, street: string) => void
}

export function useCheckoutLocation({ enabled, configured, initialTown, initialStreet, onDetected }: Options) {
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
		started.current = true
		if (!navigator.geolocation) {
			queueMicrotask(() => setStatus('Не удалось определить адрес. Выберите населённый пункт и введите адрес вручную.'))
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
					if (!active.current || takenOver.current) return
					if (response.ok && result.status === 'matched') {
						onDetected(result.town, result.street)
						setStatus('Адрес определён. Проверьте улицу и номер дома.')
					} else if (response.ok && result.status === 'unmatched')
						setStatus('Не удалось подобрать населённый пункт доставки. Выберите его вручную.')
					else setStatus('Не удалось определить адрес. Выберите населённый пункт и введите адрес вручную.')
				} catch {
					if (active.current && !takenOver.current)
						setStatus('Не удалось определить адрес. Выберите населённый пункт и введите адрес вручную.')
				}
			},
			() => {
				if (active.current && !takenOver.current)
					setStatus('Не удалось определить адрес. Выберите населённый пункт и введите адрес вручную.')
			},
			{ timeout: 10000, maximumAge: 0, enableHighAccuracy: false }
		)
		return () => {
			active.current = false
		}
	}, [enabled, configured, initialTown, initialStreet, onDetected])

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
