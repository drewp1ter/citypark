import type { ReactNode } from 'react'

/** Page title with a slot for buttons on the right (they wrap under the title on phones). */
export function PageHeader({ title, actions }: { title: string; actions?: ReactNode }) {
	return (
		<header className='mb-6 flex flex-wrap items-center justify-between gap-3'>
			<h1 className='text-2xl font-semibold'>{title}</h1>
			{actions !== undefined && <div className='flex flex-wrap items-center gap-2'>{actions}</div>}
		</header>
	)
}
