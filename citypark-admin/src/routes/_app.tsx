import { createFileRoute, Outlet, redirect } from '@tanstack/react-router'
import { getSession } from '../shared/api'
import { AppShell } from '../shared/ui/app-shell'

export const Route = createFileRoute('/_app')({
	beforeLoad: ({ location }) => {
		if (!getSession()) throw redirect({ to: '/login', search: { next: location.href } })
	},
	component: () => (
		<AppShell>
			<Outlet />
		</AppShell>
	),
})
