import { cn } from '@/shared/lib'
import { CloseIcon } from './icons'
import type { FieldTone } from './fields'

type ClearButtonProps = {
	onClick?: () => void
	/** Accessible name; Russian, shown to screen readers. */
	label?: string
	tone?: FieldTone
	className?: string
}

/** «×» that empties a text input. 44×44 hit area; the parent owns the state and passes `onClick`. */
export function ClearButton({ onClick, label = 'Очистить поле', tone = 'default', className }: ClearButtonProps) {
	return (
		<button
			type='button'
			aria-label={label}
			onClick={onClick}
			className={cn(
				'inline-flex size-11 shrink-0 cursor-pointer items-center justify-center rounded-full',
				tone === 'inverse' ? 'text-inverse-muted hover:text-inverse-text' : 'text-text-muted hover:text-text',
				className
			)}
		>
			<CloseIcon width={16} height={16} aria-hidden='true' />
		</button>
	)
}
