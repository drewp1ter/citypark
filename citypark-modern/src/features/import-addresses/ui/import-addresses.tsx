'use client'

import { useEffect, useRef, useSyncExternalStore } from 'react'
import { usePathname, useRouter } from 'next/navigation'
import { getGuestSnapshot, removeImportedGuests, useGuestAddresses, type AddressSurface } from '@/entities/address'
import { Button } from '@/shared/ui'
import { getImportAccountAction, importAddressAction } from '../api/actions'
import { createImportCoordinator } from '../model/import-state'

const coordinators = new Map<AddressSurface, ReturnType<typeof createImportCoordinator>>()
function coordinatorFor(surface: AddressSurface) {
	let coordinator = coordinators.get(surface)
	if (!coordinator) {
		coordinator = createImportCoordinator(surface, {
			guest: getGuestSnapshot,
			verify: getImportAccountAction,
			importOne: importAddressAction,
			remove: removeImportedGuests,
			storage: () => sessionStorage,
		})
		coordinators.set(surface, coordinator)
	}
	return coordinator
}

type Props = { surface: AddressSurface; initialAccountId: number | null; manual?: boolean }
export function ImportAddresses({ surface, initialAccountId, manual = false }: Props) {
	const coordinator = coordinatorFor(surface)
	const state = useSyncExternalStore(coordinator.subscribe, coordinator.getSnapshot, coordinator.getServerSnapshot)
	const guest = useGuestAddresses(state.accountId !== null)
	const router = useRouter()
	const pathname = usePathname()
	const panel = useRef<HTMLElement>(null)
	const origin = useRef<HTMLElement | null>(null)
	useEffect(() => {
		if (!manual) return coordinator.attach()
	}, [coordinator, manual])
	useEffect(() => {
		if (manual) return
		// Cookie-mutating actions re-render server content; pathname/focus/pageshow
		// also recheck persistent layouts and restored tabs, including Google returns.
		const refresh = () => {
			void coordinator.synchronize().then((accountId) => {
				if (accountId !== undefined && accountId !== initialAccountId) router.refresh()
			})
		}
		const visible = () => {
			if (document.visibilityState === 'visible') refresh()
		}
		refresh()
		window.addEventListener('focus', refresh)
		window.addEventListener('pageshow', refresh)
		document.addEventListener('visibilitychange', visible)
		return () => {
			window.removeEventListener('focus', refresh)
			window.removeEventListener('pageshow', refresh)
			document.removeEventListener('visibilitychange', visible)
		}
	}, [coordinator, manual, initialAccountId, pathname, router])
	useEffect(() => {
		if (!manual) coordinator.offer()
	}, [coordinator, manual, guest])
	useEffect(() => {
		if (!manual && state.open && state.focus) {
			origin.current = document.activeElement instanceof HTMLElement ? document.activeElement : null
			panel.current?.focus()
		}
	}, [manual, state.open, state.focus])
	if (!state.accountId) return null
	if (manual)
		return initialAccountId === state.accountId && guest.status === 'ready' && guest.book.entries.length > 0 ? (
			<Button variant='outline' disabled={state.phase === 'pending'} onClick={() => coordinator.offer(true)}>
				Добавить адреса из этого браузера
			</Button>
		) : null
	if (!state.open || (state.phase === 'offered' && (guest.status !== 'ready' || !guest.book.entries.length)))
		return null
	const pending = state.phase === 'pending'
	return (
		<section
			ref={panel}
			tabIndex={-1}
			aria-label='Импорт адресов'
			className='border-border bg-surface mx-auto my-4 flex w-full max-w-3xl flex-col items-start gap-3 rounded-lg border p-4 md:p-6'
		>
			<h2 className='text-lg font-semibold'>Добавить сохранённые адреса в аккаунт?</h2>
			<div role='status' aria-live='polite' aria-atomic='true'>
				{pending && <p>Добавляем адреса…</p>}
				{state.phase !== 'offered' && (
					<p>
						Добавлено: {state.created}. Уже сохранено: {state.skipped}.
					</p>
				)}
				{state.needsReview && <p>Некоторые адреса нужно уточнить в разделе «Мои адреса»</p>}
			</div>
			{state.phase === 'error' && (
				<div role='alert'>
					<p>Не все адреса добавлены. Оставшиеся адреса сохранены в этом браузере.</p>
					<p>{state.message}</p>
				</div>
			)}
			<div className='flex flex-wrap gap-2'>
				{state.phase !== 'success' && (
					<Button disabled={pending} onClick={() => void coordinator.start(() => router.refresh())}>
						{state.phase === 'error' ? 'Повторить' : 'Добавить адреса'}
					</Button>
				)}
				<Button
					variant='outline'
					onClick={() => {
						coordinator.dismiss()
						const target = origin.current?.isConnected
							? origin.current
							: document.querySelector<HTMLElement>('main button, main a')
						target?.focus()
					}}
				>
					{state.phase === 'success' ? 'Закрыть' : 'Не сейчас'}
				</Button>
			</div>
		</section>
	)
}
