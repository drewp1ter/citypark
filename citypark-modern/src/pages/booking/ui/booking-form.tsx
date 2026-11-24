'use client'

import Image from 'next/image'
import { startTransition, useEffect, useRef, useState } from 'react'
import { bookingCapacities, type BookingQuote, type BookingStay, type Room } from '@/entities/room/index.client'
import { addDays, formatPrice, todayInMoscow } from '@/shared/lib'
import { Button, Select, TextField, cardClass } from '@/shared/ui'
import { createReservationAction, getQuoteAction, lookupReservationAction } from '../api/actions'
import { guestSchema, type BookingState } from '../model/schema'
import { rememberAttempt, restoreAttempt, type SavedAttempt } from '../model/attempt'

const initialGuest = { name: '', email: '', phone: '' }
export function BookingForm({
	rooms,
	initialStay,
	initialKey,
	hotelPhone,
}: {
	rooms: readonly Room[]
	initialStay: BookingStay
	initialKey?: string
	hotelPhone: { display: string; href: string }
}) {
	const [stay, setStay] = useState(initialStay)
	const [guest, setGuest] = useState(initialGuest)
	const [state, setState] = useState<BookingState>(
		initialKey ? { status: 'pending', idempotencyKey: initialKey } : { status: 'idle' }
	)
	const [quote, setQuote] = useState<BookingQuote>()
	const [busy, setBusy] = useState(false)
	const [saved, setSaved] = useState<SavedAttempt>()
	const inFlight = useRef(false)
	const room = rooms.find((r) => r.id === stay.roomTypeId)
	const uncertain = state.status === 'pending' || state.status === 'unknown'
	const locked = busy || uncertain || state.status === 'confirmed'
	const key = state.idempotencyKey

	useEffect(() => {
		if (!initialKey) return
		const attempt = restoreAttempt(initialKey)
		startTransition(() => {
			if (attempt) {
				setSaved(attempt)
				setStay({
					roomTypeId: attempt.values.roomTypeId,
					checkIn: attempt.values.checkIn,
					checkOut: attempt.values.checkOut,
					guests: attempt.values.guests,
				})
				setGuest(attempt.values.guest)
				setQuote(attempt.quote)
			}
		})
	}, [initialKey])
	useEffect(() => {
		if (!uncertain || !key) return
		let disposed = false
		let querying = false
		const poll = () => {
			if (document.visibilityState !== 'visible' || querying || inFlight.current) return
			querying = true
			startTransition(async () => {
				try {
					const result = await lookupReservationAction(key)
					if (!disposed) {
						setState(result)
						if (
							result.code === 'PRICE_CHANGED' ||
							result.code === 'QUOTE_EXPIRED' ||
							result.code === 'ROOM_UNAVAILABLE'
						)
							setQuote(undefined)
					}
				} catch {
					if (!disposed)
						setState({
							status: 'unknown',
							idempotencyKey: key,
							message: 'Не удалось проверить бронирование. Продолжим проверку.',
						})
				} finally {
					querying = false
				}
			})
		}
		poll()
		const timer = setInterval(poll, 5000)
		document.addEventListener('visibilitychange', poll)
		return () => {
			disposed = true
			clearInterval(timer)
			document.removeEventListener('visibilitychange', poll)
		}
	}, [uncertain, key])

	function changeStay(change: Partial<BookingStay>) {
		setStay((current) => ({ ...current, ...change }))
		setQuote(undefined)
		setState({ status: 'idle' })
		setSaved(undefined)
	}
	async function refreshQuote() {
		if (inFlight.current || uncertain) return
		inFlight.current = true
		setBusy(true)
		try {
			const result = await getQuoteAction(stay)
			setQuote(result.quote)
			setState(result)
			setSaved(undefined)
			if (result.quote) {
				const url = new URL(location.href)
				url.searchParams.delete('attempt')
				for (const [name, value] of Object.entries(stay)) url.searchParams.set(name, String(value))
				history.replaceState(null, '', url)
			}
		} catch {
			setState({ status: 'error', message: 'Бронирование временно недоступно' })
		} finally {
			inFlight.current = false
			setBusy(false)
		}
	}
	async function submit(attempt: SavedAttempt) {
		if (inFlight.current) return
		inFlight.current = true
		setBusy(true)
		try {
			rememberAttempt(attempt)
			setSaved(attempt)
		} catch {
			setState({ status: 'error', message: 'Разрешите сохранение данных сеанса в браузере перед бронированием.' })
			inFlight.current = false
			setBusy(false)
			return
		}
		setState({ status: 'pending', idempotencyKey: attempt.request.idempotencyKey })
		try {
			const result = await createReservationAction(attempt.request, attempt.quote)
			setState(result)
			if (result.code === 'PRICE_CHANGED' || result.code === 'QUOTE_EXPIRED' || result.code === 'ROOM_UNAVAILABLE')
				setQuote(undefined)
		} catch {
			setState({
				status: 'pending',
				idempotencyKey: attempt.request.idempotencyKey,
				message: 'Ответ задерживается. Проверяем бронирование.',
			})
		} finally {
			inFlight.current = false
			setBusy(false)
		}
	}
	function book() {
		if (!quote || uncertain || busy) return
		const parsed = guestSchema.safeParse(guest)
		if (!parsed.success) {
			setState({
				status: 'error',
				message: 'Проверьте данные бронирования',
				errors: Object.fromEntries(parsed.error.issues.map((i) => [i.path[0], i.message])),
			})
			return
		}
		const values = { ...stay, guest: parsed.data }
		// Retain the request across definitive transport/pre-dispatch errors, too.
		const attempt =
			saved &&
			saved.request.quoteId === quote.quoteId &&
			JSON.stringify(saved.request.guest) === JSON.stringify(parsed.data)
				? saved
				: {
						request: { idempotencyKey: crypto.randomUUID(), quoteId: quote.quoteId, guest: parsed.data },
						values,
						quote,
					}
		void submit(attempt)
	}
	const confirmation = state.reservation
	if (confirmation)
		return (
			<section className={cardClass('surface', 'gap-4 rounded-lg p-5 md:p-8')} aria-live='polite'>
				<h2 className='heading text-2xl'>Бронирование подтверждено</h2>
				{confirmation.reservationNumber && (
					<p>Номер бронирования: {confirmation.reservationNumber}</p>
				)}
				<p>
					{rooms.find((r) => r.id === confirmation.roomTypeId)?.name} · {confirmation.checkIn} — {confirmation.checkOut}{' '}
					· Гостей: {confirmation.guests}
				</p>
				<p className='text-xl font-semibold'>{formatPrice(confirmation.price.total)} · без предоплаты</p>
				{confirmation.confirmationEmailStatus && (
					<p>
						{confirmation.confirmationEmailStatus === 'sent'
							? 'Подтверждение отправлено на электронную почту.'
							: 'Бронирование сохранено. Подтверждение отправим на электронную почту; отправка ещё ожидается.'}
					</p>
				)}
				<p>
					Для изменения или отмены позвоните в отель:{' '}
					<a className='underline' href={hotelPhone.href}>
						{hotelPhone.display}
					</a>
					.
				</p>
			</section>
		)
	return (
		<div className='grid items-start gap-6 md:grid-cols-2 md:gap-10'>
			<form
				noValidate
				className={cardClass('surface', 'gap-4 rounded-lg p-5 md:p-8')}
				onSubmit={(event) => {
					event.preventDefault()
					startTransition(() => {
						if (quote) book()
						else void refreshQuote()
					})
				}}
				aria-busy={busy}
			>
				<fieldset disabled={locked} className='grid min-w-0 gap-4'>
					<legend className='mb-3 text-xl font-semibold'>Проживание</legend>
					<Select
						label='Тип номера'
						name='roomTypeId'
						value={stay.roomTypeId}
						onValueChange={(value) => changeStay({ roomTypeId: value as BookingStay['roomTypeId'] })}
						options={rooms.map((r) => ({ value: r.id, label: r.name }))}
						error={state.errors?.roomTypeId}
					/>
					<div className='grid gap-4 sm:grid-cols-2'>
						<TextField
							label='Заезд'
							type='date'
							name='checkIn'
							min={todayInMoscow()}
							max='3999-12-30'
							value={stay.checkIn}
							onChange={(e) => changeStay({ checkIn: e.target.value })}
							error={state.errors?.checkIn}
							required
						/>
						<TextField
							label='Выезд'
							type='date'
							name='checkOut'
							min={addDays(stay.checkIn, 1)}
							max='3999-12-31'
							value={stay.checkOut}
							onChange={(e) => changeStay({ checkOut: e.target.value })}
							error={state.errors?.checkOut}
							required
						/>
					</div>
					<Select
						label='Гости'
						name='guests'
						value={String(stay.guests)}
						onValueChange={(value) => changeStay({ guests: Number(value) })}
						options={[1, 2, 3, 4].map((n) => ({ value: String(n), label: String(n) }))}
						error={state.errors?.guests}
					/>
					<p className='text-text-muted text-sm'>
						Вместимость номера — {bookingCapacities[stay.roomTypeId]} чел. Один номер на весь срок, без предоплаты.
					</p>
				</fieldset>
				<fieldset disabled={locked} className='grid min-w-0 gap-4'>
					<legend className='mb-3 text-xl font-semibold'>Данные гостя</legend>
					<p className='text-text-muted text-sm'>Все поля в этом разделе обязательны.</p>
					<TextField
						label='ФИО'
						name='name'
						autoComplete='name'
						value={guest.name}
						onChange={(e) => setGuest((g) => ({ ...g, name: e.target.value }))}
						error={state.errors?.name}
						required
					/>
					<TextField
						label='Электронная почта (необязательно)'
						name='email'
						type='email'
						autoComplete='email'
						value={guest.email}
						onChange={(e) => setGuest((g) => ({ ...g, email: e.target.value }))}
						error={state.errors?.email}
					/>
					<TextField
						label='Телефон'
						name='phone'
						type='tel'
						autoComplete='tel'
						value={guest.phone}
						onChange={(e) => setGuest((g) => ({ ...g, phone: e.target.value }))}
						error={state.errors?.phone}
						required
					/>
				</fieldset>
				{state.message && (
					<p role='alert' className='text-text'>
						{state.message}
					</p>
				)}
				{uncertain ? (
					<div aria-live='polite' className='grid gap-3'>
						<h2 className='text-xl font-semibold'>Проверяем бронирование</h2>
						<p>Не создавайте новое бронирование. Проверяем результат каждые пять секунд, пока эта страница открыта.</p>
						<p className='text-text-muted text-sm break-all'>Ключ запроса: {key}</p>
						{saved && (
							<Button
								disabled={busy}
								onClick={() =>
									startTransition(() => {
										void submit(saved)
									})
								}
								variant='outline'
							>
								Повторить тот же запрос
							</Button>
						)}
					</div>
				) : (
					<>
						{quote && (
							<div aria-live='polite' className='grid gap-2'>
								<p>
									{quote.price.nights} ноч. × {formatPrice(quote.price.pricePerNight)}
								</p>
								<p className='text-2xl font-semibold'>Итого: {formatPrice(quote.price.total)}</p>
								<p className='text-text-muted text-sm'>
									Нажимая «Забронировать», вы подтверждаете эту стоимость. Подтверждение отправим на электронную почту.
								</p>
							</div>
						)}
						<Button type='submit' disabled={busy} size='xl'>
							{busy ? 'Подождите…' : quote ? 'Забронировать' : 'Проверить наличие и стоимость'}
						</Button>
						{quote && (
							<Button
								disabled={busy}
								onClick={() =>
									startTransition(() => {
										void refreshQuote()
									})
								}
								variant='outline'
							>
								Обновить стоимость
							</Button>
						)}
					</>
				)}
			</form>
			{room && (
				<aside className={cardClass('surface', 'gap-4 overflow-hidden rounded-lg p-3 md:p-5')}>
					<div className='relative h-60 overflow-hidden rounded-md'>
						<Image
							src={room.photo.src}
							alt={room.photo.alt}
							fill
							sizes='(min-width: 768px) 50vw, 100vw'
							className='object-cover'
						/>
					</div>
					<h2 className='heading text-2xl'>{room.name}</h2>
					<p>{room.areaDetailed ?? room.area}</p>
					{room.summary && <p>{room.summary}</p>}
					<p className='text-text-muted'>Наличие номера окончательно подтверждается при бронировании.</p>
				</aside>
			)}
		</div>
	)
}
