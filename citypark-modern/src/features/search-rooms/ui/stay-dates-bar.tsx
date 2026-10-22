'use client'

import Form from 'next/form'
import { useState } from 'react'
import { addDays, cn, useToday } from '@/shared/lib'
import { Button, Select } from '@/shared/ui'

type StayDatesBarProps = {
	/**
	 * "inline": a small pill with two dates and a button (home page).
	 * "rooms": the search bar of the rooms page: with guests, two date tiles + a wide button on mobile.
	 */
	variant?: 'inline' | 'rooms'
	submitLabel?: string
	/** Values from the address bar (ISO dates, "1" | "2" | "3-4"); today / tomorrow / 2 when missing. */
	defaultCheckIn?: string
	defaultCheckOut?: string
	defaultGuests?: string
	/** Other address parameters to keep when searching (the active filters of the rooms page). */
	preserve?: Record<string, string>
	className?: string
}

const guestsOptions = [
	{ value: '1', label: '1' },
	{ value: '2', label: '2' },
	{ value: '3-4', label: '3–4' },
]

const guestsLabel: Record<string, string> = { '1': '1 гость', '2': '2 гостя', '3-4': '3–4 гостя' }

/**
 * Check-in / check-out bar. It is a plain GET form to /rooms?checkIn=…&checkOut=…&guests=…, so it works without
 * JavaScript and the rooms page reads everything from the URL.
 */
export function StayDatesBar({
	variant = 'inline',
	submitLabel = 'Проверить',
	defaultCheckIn,
	defaultCheckOut,
	defaultGuests = '2',
	preserve,
	className,
}: StayDatesBarProps) {
	const rooms = variant === 'rooms'
	const today = useToday()
	const [checkIn, setCheckIn] = useState<string>()
	const [checkOut, setCheckOut] = useState<string>()

	// The visitor's own choice wins, then the address bar, then today / tomorrow. Check-out is always after check-in.
	const checkInValue = checkIn ?? defaultCheckIn ?? today
	const earliestCheckOut = addDays(checkInValue, 1)
	const chosenCheckOut = checkOut ?? defaultCheckOut
	const checkOutValue = chosenCheckOut && chosenCheckOut >= earliestCheckOut ? chosenCheckOut : earliestCheckOut

	const label = cn(
		'flex items-center gap-2 px-3 text-sm text-text-muted',
		rooms &&
			'max-md:flex-col max-md:items-stretch max-md:gap-0 max-md:rounded-xs max-md:bg-bg max-md:px-3 max-md:pt-1.5 max-md:pb-0.5 max-md:text-xs md:px-3.5'
	)
	const input = cn(
		'border-0 bg-transparent font-[inherit] text-text',
		rooms ? 'min-h-11 max-md:text-[15px]' : 'min-h-11'
	)

	return (
		<Form
			action='/rooms'
			id={rooms ? 'search' : undefined}
			className={cn(
				rooms
					? 'border-border bg-surface md:rounded-pill grid scroll-mt-24 grid-cols-2 gap-1.5 rounded-lg border p-2 md:flex md:flex-wrap md:p-1.5'
					: 'rounded-pill border-border bg-surface flex flex-wrap gap-2 border p-1.5',
				className
			)}
		>
			{Object.entries(preserve ?? {}).map(([name, value]) => (
				<input key={name} type='hidden' name={name} value={value} />
			))}
			<label className={label}>
				Заезд
				<input
					type='date'
					name='checkIn'
					min={today || undefined}
					value={checkInValue}
					onChange={(event) => setCheckIn(event.target.value)}
					className={input}
				/>
			</label>
			<label className={cn(label, rooms && 'md:border-border md:border-l')}>
				Выезд
				<input
					type='date'
					name='checkOut'
					min={earliestCheckOut || undefined}
					value={checkOutValue}
					onChange={(event) => setCheckOut(event.target.value)}
					className={input}
				/>
			</label>
			{rooms && (
				// Hidden on mobile (the button says "2 гостя"), but still sent with the form.
				<Select
					label='Гости'
					name='guests'
					options={guestsOptions}
					defaultValue={defaultGuests}
					className={cn(
						label,
						'md:border-border font-normal max-md:hidden md:flex-row md:border-l',
						'[&_button]:text-text [&_button]:min-h-11 [&_button]:w-auto [&_button]:border-0 [&_button]:bg-transparent [&_button]:px-0 [&_button]:py-0'
					)}
				/>
			)}
			<Button
				type='submit'
				size={rooms ? 'xl' : 'md'}
				className={cn('px-5', rooms && 'col-span-2 md:col-auto md:min-h-12 md:px-[22px]')}
			>
				{rooms ? (
					<>
						<span className='md:hidden'>
							{submitLabel} · {guestsLabel[defaultGuests] ?? guestsLabel['2']}
						</span>
						<span className='max-md:hidden'>{submitLabel}</span>
					</>
				) : (
					submitLabel
				)}
			</Button>
		</Form>
	)
}
