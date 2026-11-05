'use client'

import { useContext, useEffect, useRef, useState, type ReactNode } from 'react'
import {
	ComboBox,
	ComboBoxStateContext,
	Input,
	Label,
	ListBox,
	ListBoxItem,
	Popover,
	Text,
} from 'react-aria-components'
import { cn } from '@/shared/lib'
import { TextField, controlClass, fieldErrorClass, fieldLabelClass } from '@/shared/ui'

const MIN_QUERY = 3
const DEBOUNCE_MS = 700

type AddressFieldProps = {
	label: ReactNode
	name: string
	/** The selected «Населённый пункт». */
	town: string
	/** False → a plain text field, no requests. */
	suggestions: boolean
	/** The value after a server error. */
	defaultValue?: string
	value: string
	onValueChange: (value: string) => void
	onManualEdit: () => void
	error?: string
	className?: string
}

const escapeRegExp = (text: string) => text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

/** Wraps the words the customer typed in <mark>. */
function Highlighted({ text, query }: { text: string; query: string }) {
	const words = query.split(/[\s,]+/).filter(Boolean)
	if (words.length === 0) return text
	const parts = text.split(new RegExp(`(${words.map(escapeRegExp).join('|')})`, 'gi'))
	return parts.map((part, index) =>
		index % 2 === 1 ? (
			<mark key={index} className='bg-accent text-on-accent rounded-[4px] px-0.5 font-semibold'>
				{part}
			</mark>
		) : (
			part
		)
	)
}

/** Results arrive after the keystroke, when React Aria has already decided not to open the list: open it once they are here. */
function OpenWhenReady({ count, input }: { count: number; input: { current: HTMLInputElement | null } }) {
	const state = useContext(ComboBoxStateContext)
	// `state` changes on every render; only a change of `count` should open the list (Esc must keep it closed).
	const opened = useRef(0)
	useEffect(() => {
		if (count === opened.current) return
		opened.current = count
		if (count > 0 && document.activeElement === input.current) state?.open(undefined, 'input')
	}, [count, input, state])
	return null
}

/** «Улица и дом» with up to 5 DaData suggestions (through our own route). Any typed text is still accepted. */
export function AddressField({
	label,
	name,
	town,
	suggestions,
	value,
	onValueChange,
	onManualEdit,
	error,
	className,
}: AddressFieldProps) {
	if (!suggestions) {
		return (
			<TextField
				label={label}
				name={name}
				autoComplete='street-address'
				placeholder='Начните вводить адрес'
				value={value}
				onChange={(event) => {
					onManualEdit()
					onValueChange(event.target.value)
				}}
				error={error}
				className={className}
			/>
		)
	}
	return (
		<SuggestingField
			label={label}
			name={name}
			town={town}
			value={value}
			onValueChange={onValueChange}
			onManualEdit={onManualEdit}
			error={error}
			className={className}
		/>
	)
}

function SuggestingField({
	label,
	name,
	town,
	value,
	onValueChange,
	onManualEdit,
	error,
	className,
}: Omit<AddressFieldProps, 'suggestions'>) {
	const [items, setItems] = useState<string[]>([])
	const inputRef = useRef<HTMLInputElement>(null)
	// Set when the text comes from a chosen item, so picking it does not fetch suggestions again.
	const picked = useRef(false)

	useEffect(() => {
		const query = value.trim()
		if (picked.current || !town || query.length < MIN_QUERY) {
			picked.current = false
			setItems([])
			return
		}
		const controller = new AbortController()
		const timer = setTimeout(async () => {
			try {
				const response = await fetch(`/api/address-suggest?${new URLSearchParams({ q: query, town })}`, {
					signal: controller.signal,
				})
				if (!response.ok) return setItems([])
				setItems((await response.json()) as string[])
			} catch {
				// Aborted or offline: no list, the field keeps working as a plain input.
				if (!controller.signal.aborted) setItems([])
			}
		}, DEBOUNCE_MS)
		return () => {
			clearTimeout(timer)
			controller.abort()
		}
	}, [value, town])

	return (
		<ComboBox
			name={name}
			allowsCustomValue
			menuTrigger='input'
			// DaData already matched the query; a local "contains" filter would hide «ул Гоголя, д 6» for «Гоголя 6».
			defaultFilter={() => true}
			inputValue={value}
			onInputChange={onValueChange}
			onSelectionChange={(key) => {
				if (key === null) return
				picked.current = true
				onValueChange(String(key))
				setItems([])
				// Caret to the end, so the customer can add «, корп. 2».
				requestAnimationFrame(() => {
					const input = inputRef.current
					input?.setSelectionRange(input.value.length, input.value.length)
				})
			}}
			items={items.map((value) => ({ id: value }))}
			isInvalid={Boolean(error)}
			className={cn(fieldLabelClass, className)}
		>
			<Label>{label}</Label>
			<OpenWhenReady count={items.length} input={inputRef} />
			<Input
				ref={inputRef}
				onChange={onManualEdit}
				autoComplete='off'
				placeholder='Начните вводить адрес'
				className={cn(
					controlClass,
					'data-[focus-visible]:outline-accent data-[focus-visible]:outline-2 data-[focus-visible]:outline-offset-2',
					'data-[invalid]:border-danger'
				)}
			/>
			{error && (
				<Text slot='errorMessage' role='alert' className={fieldErrorClass}>
					{error}
				</Text>
			)}
			<Popover
				offset={6}
				className={cn(
					'max-h-72 w-(--trigger-width) overflow-auto rounded-xs border p-1 text-base font-normal',
					'border-border bg-surface text-text',
					'data-[entering]:motion-safe:animate-[select-in_120ms_ease-out] data-[exiting]:motion-safe:animate-[select-out_90ms_ease-in]'
				)}
			>
				<p className='text-text-muted px-3 py-2 text-sm'>Выберите вариант или продолжите ввод</p>
				<ListBox className='outline-none'>
					{(item: { id: string }) => (
						<ListBoxItem
							id={item.id}
							textValue={item.id}
							className={cn(
								'flex min-h-11 cursor-pointer items-center rounded-[12px] px-3',
								'data-[focused]:bg-bg',
								'data-[focus-visible]:outline-accent data-[focus-visible]:outline-2 data-[focus-visible]:-outline-offset-2'
							)}
						>
							<span>
								<Highlighted text={item.id} query={value} />
							</span>
						</ListBoxItem>
					)}
				</ListBox>
			</Popover>
		</ComboBox>
	)
}
