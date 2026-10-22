import { site } from '@/shared/config'
import { cn } from '@/shared/lib'

/** The «Сити Парк» logo, drawn in `currentColor`. Size it with a height class; the width follows the aspect ratio. */
export function Logo({ className }: { className?: string }) {
	return (
		<span
			role='img'
			aria-label={site.wordmark}
			className={cn(
				'inline-block aspect-[1624/1363] bg-current [mask:url(/brand/logo.svg)_center/contain_no-repeat] forced-colors:bg-[CanvasText]',
				className
			)}
		/>
	)
}
