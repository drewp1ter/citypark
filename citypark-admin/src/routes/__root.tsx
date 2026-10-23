import { QueryClientProvider } from '@tanstack/react-query'
import { Toast } from '@heroui/react'
import { createRootRoute, Outlet } from '@tanstack/react-router'
import { queryClient } from '../router'

export const Route = createRootRoute({
	component: RootLayout,
})

function RootLayout() {
	return (
		<QueryClientProvider client={queryClient}>
			<Outlet />
			<Toast.Provider />
		</QueryClientProvider>
	)
}
