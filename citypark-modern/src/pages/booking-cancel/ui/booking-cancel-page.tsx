'use client'

import { useEffect, useState, useTransition } from 'react'

import { getCancellation, submitCancellation } from '../api/cancellation'

export function BookingCancelPage() {
	const [result, setResult] = useState<Awaited<ReturnType<typeof getCancellation>>>()
	const [busy, startTransition] = useTransition()
	const token = () => window.location.hash.slice(1)

	useEffect(() => {
		let active = true
		void getCancellation(window.location.hash.slice(1)).then((value) => {
			if (active) setResult(value)
		}).catch(() => {
			if (active) setResult({ status: 'unavailable' })
		})
		return () => { active = false }
	}, [])

	function update(confirm: boolean) {
		startTransition(async () => {
			try {
				setResult(await (confirm ? submitCancellation : getCancellation)(token()))
			} catch {
				setResult({ status: confirm ? 'pending' : 'unavailable' })
			}
		})
	}

	const status = result?.status
	const message = {
		eligible: 'Подтвердите отмену бронирования.',
		canceled: 'Бронирование отменено.',
		closed: 'Онлайн-отмена недоступна. Свяжитесь с гостиницей.',
		invalid: 'Ссылка на отмену бронирования недействительна.',
		unavailable: 'Не удалось проверить бронирование. Попробуйте ещё раз.',
		pending: 'Гостиница ещё не подтвердила результат. Проверьте статус бронирования.',
	}

	return (
		<section className="mx-auto max-w-2xl px-4 py-12">
			<h1 className="font-display text-2xl">Отмена бронирования</h1>
			{result && 'reservationNumber' in result && (
				<p className="mt-4">Бронирование № {result.reservationNumber}, {result.checkIn} — {result.checkOut}</p>
			)}
			<p className="mt-4" role="status">{busy || !status ? 'Проверяем бронирование…' : message[status]}</p>
			{(status === 'eligible' || status === 'pending' || status === 'unavailable') && (
				<button
					type="button"
					className="mt-4 min-h-11 rounded-lg bg-accent px-4 text-on-accent"
					disabled={busy}
					onClick={() => update(status === 'eligible')}
				>
					{status === 'eligible' ? 'Отменить бронирование' : 'Проверить статус'}
				</button>
			)}
			{(status === 'closed' || status === 'invalid' || status === 'unavailable' || status === 'pending') && (
				<a className="mt-4 block underline" href="tel:+79183119710">8 918 311-97-10</a>
			)}
		</section>
	)
}
