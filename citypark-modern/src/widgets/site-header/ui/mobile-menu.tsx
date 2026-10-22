'use client'

import { useRef, type MouseEvent } from 'react'
import { ThemeToggle } from './theme-toggle'
import { site } from '@/shared/config'
import { AppLink, Button, CloseIcon, IconButton, Logo, MenuIcon } from '@/shared/ui'

/**
 * Menu button + full-screen sheet (not in the design; built from the existing tokens).
 * It is a native modal <dialog>, so the browser gives us the focus trap, Esc to close and an inert page behind it.
 */
export function MobileMenu() {
	const dialog = useRef<HTMLDialogElement>(null)

	const closeOnLinkClick = (event: MouseEvent) => {
		if ((event.target as HTMLElement).closest('a')) dialog.current?.close()
	}

	return (
		<>
			<IconButton variant='inverse' aria-label='Открыть меню' onClick={() => dialog.current?.showModal()}>
				<MenuIcon />
			</IconButton>

			<dialog
				ref={dialog}
				aria-label='Меню'
				onClick={closeOnLinkClick}
				className='bg-bg text-text fixed inset-0 m-0 size-full max-h-none max-w-none overflow-y-auto p-0 backdrop:bg-transparent open:flex open:flex-col'
			>
				<div className='flex items-center justify-between px-3 pt-2.5'>
					<div className='rounded-pill border-border bg-surface flex w-full items-center justify-between border py-1.5 pr-1.5 pl-3'>
						<AppLink href='/' className='inline-flex min-h-11 items-center'>
							<Logo className='h-10' />
						</AppLink>
						<IconButton variant='inverse' aria-label='Закрыть меню' onClick={() => dialog.current?.close()}>
							<CloseIcon />
						</IconButton>
					</div>
				</div>

				<nav aria-label='Основное меню' className='flex flex-col px-6 pt-6'>
					{site.nav.map(({ href, label }) => (
						<AppLink
							key={href}
							href={href}
							className='border-divider font-display flex min-h-14 items-center border-b text-[28px] leading-none font-medium tracking-[-0.03em]'
						>
							{label}
						</AppLink>
					))}
				</nav>

				<div className='mt-auto flex flex-col gap-4 px-6 pt-8 pb-6'>
					<ul className='flex flex-col'>
						{[site.phones.restaurant, site.phones.hotel, site.phones.free].map((phone) => (
							<li key={phone.href}>
								<AppLink href={phone.href} className='border-divider flex justify-between gap-3 border-b py-3.5'>
									<span className='text-text-muted'>{phone.shortLabel}</span>
									<strong>{phone.display}</strong>
								</AppLink>
							</li>
						))}
					</ul>
					<div className='flex items-center justify-between gap-3'>
						<span className='text-text-muted'>Тема оформления</span>
						<ThemeToggle />
					</div>
					<Button href='/rooms' size='xl'>
						Забронировать
					</Button>
				</div>
			</dialog>
		</>
	)
}
