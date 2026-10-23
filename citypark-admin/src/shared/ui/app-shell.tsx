import { Button, Drawer, useOverlayState } from '@heroui/react'
import { Link, useNavigate } from '@tanstack/react-router'
import { useSyncExternalStore, type ReactNode } from 'react'
import { clearSession, getSession, onSessionChange } from '../api'
import { contentNavItems, mainNavItems, tailNavItems, type NavItem } from './nav-items'
import { ThemeToggle } from './theme-toggle'

const linkClass = 'block rounded-lg px-3 py-2 text-sm text-foreground hover:bg-default'
const activeLinkClass = 'bg-accent font-medium text-accent-foreground hover:bg-accent'

function NavLinks({ items, onNavigate }: { items: NavItem[]; onNavigate?: () => void }) {
	return (
		<ul className='flex flex-col gap-1'>
			{items.map((item) => (
				<li key={item.to}>
					<Link to={item.to} className={linkClass} activeProps={{ className: activeLinkClass }} onClick={onNavigate}>
						{item.label}
					</Link>
				</li>
			))}
		</ul>
	)
}

function Navigation({ onNavigate }: { onNavigate?: () => void }) {
	return (
		<nav aria-label='Разделы' className='flex flex-col gap-4'>
			<NavLinks items={mainNavItems} onNavigate={onNavigate} />
			<div>
				<h2 className='px-3 pb-1 text-xs font-semibold text-muted uppercase'>Контент</h2>
				<NavLinks items={contentNavItems} onNavigate={onNavigate} />
			</div>
			<NavLinks items={tailNavItems} onNavigate={onNavigate} />
		</nav>
	)
}

function UserBlock() {
	const email = useSyncExternalStore(onSessionChange, () => getSession()?.email ?? '')
	const navigate = useNavigate()

	function signOut() {
		clearSession()
		void navigate({ to: '/login' })
	}

	return (
		<div className='flex flex-col gap-3'>
			<p className='truncate text-sm text-muted' title={email}>
				{email}
			</p>
			<div className='flex items-center gap-2'>
				<ThemeToggle />
				<Button variant='outline' className='flex-1' onPress={signOut}>
					Выйти
				</Button>
			</div>
		</div>
	)
}

function MenuIcon() {
	return (
		<svg
			width='20'
			height='20'
			viewBox='0 0 24 24'
			fill='none'
			stroke='currentColor'
			strokeWidth='2'
			aria-hidden='true'
		>
			<path d='M4 6h16M4 12h16M4 18h16' />
		</svg>
	)
}

export function AppShell({ children }: { children: ReactNode }) {
	const drawer = useOverlayState()

	return (
		<div className='min-h-screen bg-background text-foreground'>
			<aside className='fixed inset-y-0 left-0 hidden w-60 flex-col gap-6 overflow-y-auto border-r border-border bg-surface p-4 lg:flex'>
				<p className='px-3 text-lg font-semibold'>Сити Парк</p>
				<div className='flex-1'>
					<Navigation />
				</div>
				<UserBlock />
			</aside>

			<header className='sticky top-0 z-10 flex items-center gap-3 border-b border-border bg-surface px-4 py-2 lg:hidden'>
				<Button isIconOnly variant='outline' aria-label='Открыть меню' onPress={drawer.open}>
					<MenuIcon />
				</Button>
				<p className='text-lg font-semibold'>Сити Парк</p>
			</header>

			<Drawer state={drawer}>
				<Drawer.Backdrop>
					<Drawer.Content placement='left'>
						<Drawer.Dialog aria-label='Меню'>
							<Drawer.CloseTrigger />
							<Drawer.Header>
								<Drawer.Heading>Сити Парк</Drawer.Heading>
							</Drawer.Header>
							<Drawer.Body>
								<Navigation onNavigate={drawer.close} />
							</Drawer.Body>
							<Drawer.Footer>
								<UserBlock />
							</Drawer.Footer>
						</Drawer.Dialog>
					</Drawer.Content>
				</Drawer.Backdrop>
			</Drawer>

			<main className='p-4 lg:ml-60 lg:p-8'>{children}</main>
		</div>
	)
}
