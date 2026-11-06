'use client'

import { useEffect, useRef, useState, useTransition } from 'react'
import { useRouter } from 'next/navigation'
import {
	useGuestAddresses,
	saveGuestAddress,
	deleteGuestAddress,
	setGuestDefault,
	resetGuestAddresses,
	addressFieldsSchema,
	type AddressFields,
} from '@/entities/address'
import { Button } from '@/shared/ui'
import { saveAddressAction, deleteAddressAction, defaultAddressAction } from '../api/actions'
import type { AddressEntry, ManagementProps, ManagementResult } from '../model/types'
import { AddressEditor } from './address-editor'
import { AddressCard } from './address-card'

const empty: AddressFields = {
	name: '',
	recipientName: '',
	recipientPhone: '',
	city: '',
	street: '',
	apartment: '',
	entrance: '',
}
export function AddressBook({
	context,
	initialEntries,
	towns,
	suggestionsEnabled,
	loadError,
	townsError,
}: ManagementProps) {
	const guest = useGuestAddresses()
	const router = useRouter()
	const [accountEntries, setAccountEntries] = useState(initialEntries)
	const [previousEntries, setPreviousEntries] = useState(initialEntries)
	if (previousEntries !== initialEntries) {
		setPreviousEntries(initialEntries)
		setAccountEntries(initialEntries)
	}
	const entries: AddressEntry[] = context.kind === 'guest' ? guest.book.entries : accountEntries
	const [editing, setEditing] = useState<AddressEntry | 'new' | null>(null)
	const [values, setValues] = useState(empty)
	const [errors, setErrors] = useState<Record<string, string>>()
	const [message, setMessage] = useState('')
	const [authRequired, setAuthRequired] = useState(false)
	const [pending, startTransition] = useTransition()
	const busy = useRef(false)
	const active = useRef(true)
	const add = useRef<HTMLDivElement>(null)
	const origin = useRef<HTMLElement | null>(null)
	const confirmation = useRef<HTMLDialogElement>(null)
	const [removing, setRemoving] = useState<AddressEntry | 'reset' | null>(null)
	useEffect(() => {
		active.current = true
		return () => {
			active.current = false
		}
	}, [])
	const focusRequested = useRef(false)
	const focusBack = () => {
		focusRequested.current = true
	}
	useEffect(() => {
		if (pending || !focusRequested.current) return
		focusRequested.current = false
		if (origin.current?.isConnected) origin.current.focus()
		else add.current?.querySelector('button')?.focus()
	}, [pending, editing, removing])
	const finish = (result: ManagementResult, success: string, closeEditor = false) => {
		if (!active.current) return
		if (!result.ok) {
			setMessage(result.message)
			setErrors(result.fields)
			setAuthRequired(Boolean(result.authRequired))
			return
		}
		setMessage(success)
		setErrors(undefined)
		if (closeEditor) {
			setEditing(null)
			focusBack()
		}
	}
	const run = (operation: () => Promise<void>) => {
		if (busy.current) return
		busy.current = true
		startTransition(async () => {
			try {
				await operation()
			} catch {
				if (active.current) setMessage('Не удалось сохранить адрес. Попробуйте ещё раз.')
			} finally {
				busy.current = false
			}
		})
	}
	const input = (entry?: AddressEntry) => ({
		surface: context.surface,
		expectedAccountId: context.kind === 'account' ? context.accountId : 0,
		id: entry && typeof entry.id === 'number' ? entry.id : undefined,
	})
	const save = () => {
		const parsed = addressFieldsSchema.safeParse(values)
		if (!parsed.success) {
			const fields: Record<string, string> = {}
			for (const issue of parsed.error.issues) fields[String(issue.path[0])] ??= issue.message
			setErrors(fields)
			setMessage('Проверьте адрес')
			return
		}
		run(async () => {
			const result =
				context.kind === 'guest'
					? await saveGuestAddress(parsed.data, editing && editing !== 'new' ? String(editing.id) : undefined)
					: await saveAddressAction(input(editing && editing !== 'new' ? editing : undefined), parsed.data)
			if (!active.current) return
			if (result.ok && context.kind === 'account' && result.entry && typeof result.entry.id === 'number') {
				const entry = result.entry as (typeof initialEntries)[number]
				setAccountEntries((previous) =>
					[...previous.filter((item) => item.id !== entry.id), entry].sort((a, b) => a.id - b.id)
				)
			}
			finish(result as ManagementResult, 'Адрес сохранён', true)
		})
	}
	const edit = (entry: AddressEntry | 'new') => {
		origin.current = document.activeElement as HTMLElement
		setEditing(entry)
		setErrors(undefined)
		setMessage('')
		setValues(
			entry === 'new'
				? empty
				: {
						name: entry.name,
						recipientName: entry.recipientName,
						recipientPhone: entry.recipientPhone,
						city: entry.city,
						street: 'needsReview' in entry && entry.needsReview ? '' : entry.street,
						apartment: entry.apartment,
						entrance: entry.entrance,
					}
		)
	}
	const confirm = (entry: AddressEntry | 'reset') => {
		origin.current = document.activeElement as HTMLElement
		setRemoving(entry)
		confirmation.current?.showModal()
	}
	const remove = () =>
		run(async () => {
			if (!removing) return
			const result =
				removing === 'reset'
					? await resetGuestAddresses()
					: context.kind === 'guest'
						? await deleteGuestAddress(String(removing.id))
						: await deleteAddressAction(input(removing))
			if (!active.current) return
			if (result.ok) {
				if (removing !== 'reset') setAccountEntries((previous) => previous.filter((entry) => entry.id !== removing.id))
				confirmation.current?.close()
				setRemoving(null)
				focusBack()
			}
			finish(result as ManagementResult, 'Адрес удалён')
		})
	const makeDefault = (entry: AddressEntry) =>
		run(async () => {
			const result =
				context.kind === 'guest' ? await setGuestDefault(String(entry.id)) : await defaultAddressAction(input(entry))
			if (!active.current) return
			if (result.ok)
				setAccountEntries((previous) => previous.map((item) => ({ ...item, isDefault: item.id === entry.id })))
			finish(result as ManagementResult, 'Основной адрес изменён')
		})
	const retry = (
		<Button variant='outline' onClick={() => router.refresh()}>
			Попробовать ещё раз
		</Button>
	)
	if (loadError)
		return (
			<div className='flex flex-col items-start gap-3'>
				<p role='alert'>{loadError}</p>
				{retry}
			</div>
		)
	if (context.kind === 'auth-required')
		return (
			<div className='flex flex-col items-start gap-3'>
				<p role='alert'>Войдите повторно, чтобы сохранить адрес</p>
				<Button href={context.surface === 'mini' ? '/mini' : '/account/sign-in?returnTo=%2Faccount%2Faddresses'}>
					{context.surface === 'mini' ? 'Открыть приложение заново' : 'Войти'}
				</Button>
				{retry}
			</div>
		)
	const ready = context.kind !== 'guest' || guest.status === 'ready'
	return (
		<div className='flex min-w-0 flex-col gap-5'>
			{context.kind === 'guest' && <p>Адреса сохраняются только в этом браузере</p>}
			{townsError && (
				<div>
					<p role='alert'>Не удалось загрузить населённые пункты. Попробуйте ещё раз.</p>
					{retry}
				</div>
			)}
			<p role='status' aria-live='polite' className='empty:hidden'>
				{message}
			</p>
			{authRequired && (
				<Button href={context.surface === 'mini' ? '/mini' : '/account/sign-in?returnTo=%2Faccount%2Faddresses'}>
					Войдите повторно, чтобы сохранить адрес
				</Button>
			)}
			{context.kind === 'guest' && guest.status === 'loading' && <p role='status'>Загружаем адреса…</p>}
			{context.kind === 'guest' && guest.status === 'corrupt' && (
				<div>
					<p role='alert'>Не удалось прочитать сохранённые адреса</p>
					<Button variant='outline' onClick={() => confirm('reset')}>
						Сбросить сохранённые адреса
					</Button>
				</div>
			)}
			{context.kind === 'guest' && guest.status === 'unsupported' && (
				<p role='alert'>Обновите страницу, чтобы открыть сохранённые адреса</p>
			)}
			{context.kind === 'guest' && guest.status === 'unavailable' && (
				<p role='alert'>Не удалось сохранить адрес в этом браузере</p>
			)}
			{ready && entries.length === 0 && <p>У вас пока нет сохранённых адресов</p>}
			<ul className='flex min-w-0 flex-col gap-3'>
				{entries.map((entry) => (
					<AddressCard
						key={entry.id}
						entry={entry}
						unavailable={!townsError && !towns.some((town) => town.name === entry.city)}
						pending={pending || authRequired}
						onEdit={() => edit(entry)}
						onDelete={() => confirm(entry)}
						onDefault={() => makeDefault(entry)}
					/>
				))}
			</ul>
			{editing && (
				<div className='flex flex-col gap-3'>
					{editing !== 'new' && 'needsReview' in editing && editing.needsReview && (
						<>
							<p>Проверьте и уточните адрес</p>
							<p className='[overflow-wrap:anywhere]'>{editing.address}</p>
						</>
					)}
					<AddressEditor
						values={values}
						onChange={setValues}
						towns={towns}
						suggestionsEnabled={suggestionsEnabled}
						errors={errors}
						pending={pending || authRequired}
						onSave={save}
						onCancel={() => {
							setEditing(null)
							focusBack()
						}}
					/>
				</div>
			)}
			{!editing && (
				<div ref={add}>
					<Button
						className='self-start'
						disabled={!ready || pending || authRequired || townsError}
						onClick={() => edit('new')}
					>
						Добавить адрес
					</Button>
				</div>
			)}
			<dialog
				ref={confirmation}
				aria-label={removing === 'reset' ? 'Сбросить сохранённые адреса?' : 'Удалить этот адрес?'}
				onCancel={(event) => {
					if (pending) event.preventDefault()
				}}
				onClose={() => {
					setRemoving(null)
					focusBack()
				}}
				className='border-border bg-surface text-text m-auto w-[calc(100%-2rem)] max-w-md rounded-lg border p-6 backdrop:bg-[var(--scrim)]'
			>
				<p className='mb-5 text-lg font-semibold'>
					{removing === 'reset' ? 'Сбросить сохранённые адреса?' : 'Удалить этот адрес?'}
				</p>
				<div className='flex flex-wrap gap-3'>
					<Button variant='outline' autoFocus disabled={pending} onClick={() => confirmation.current?.close()}>
						Отмена
					</Button>
					<Button disabled={pending} onClick={remove}>
						Удалить
					</Button>
				</div>
			</dialog>
		</div>
	)
}
