import type { ReactNode } from 'react'
import { cn } from '@/shared/lib'

type BadgeProps = {
	children: ReactNode
	/** Show the dark dot: always, or only from 768 px ("md"). */
	dot?: boolean | 'md'
	className?: string
}

/** Small lime pill, e.g. "Белореченск · ул. Гоголя, 61". */
export function Badge({ children, dot, className }: BadgeProps) {
	return (
		<p
			className={cn(
				'rounded-pill bg-accent text-on-accent inline-flex items-center gap-2 self-start px-3 py-[5px] text-[13px] font-semibold md:px-3.5 md:py-1.5 md:text-sm',
				className
			)}
		>
			{dot && (
				<span
					aria-hidden='true'
					className={cn('bg-on-accent size-2 rounded-full', dot === 'md' && 'hidden md:inline-block')}
				/>
			)}
			{children}
		</p>
	)
}
