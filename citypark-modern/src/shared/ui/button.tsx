import type { ComponentPropsWithoutRef } from 'react'
import { cn } from '@/shared/lib'
import { AppLink } from './app-link'

export type ButtonVariant = 'accent' | 'inverse' | 'outline' | 'surface' | 'outline-inverse' | 'ghost'
export type ButtonSize = 'md' | 'lg' | 'xl'

const variants: Record<ButtonVariant, string> = {
	accent: 'bg-accent text-on-accent',
	inverse: 'bg-inverse-bg text-inverse-text',
	outline: 'border border-border-strong text-text',
	surface: 'border border-border bg-surface text-text',
	// For use on an inverse block (contacts block, featured room).
	'outline-inverse': 'border border-inverse-border-strong text-inverse-text',
	ghost: 'text-text',
}

// 44 px is the minimum touch target; lg/xl are the taller buttons from the design (48 / 52 px).
const sizes: Record<ButtonSize, string> = {
	md: 'min-h-11 px-[22px]',
	lg: 'min-h-12 px-[22px]',
	xl: 'min-h-[52px] px-6',
}

type OwnProps = { variant?: ButtonVariant; size?: ButtonSize }
type ButtonAsLink = OwnProps & { href: string } & Omit<ComponentPropsWithoutRef<'a'>, 'href'>
type ButtonAsButton = OwnProps & { href?: undefined } & ComponentPropsWithoutRef<'button'>

/** Pill button. Renders a link when `href` is given, otherwise a <button>. */
export function Button(props: ButtonAsLink | ButtonAsButton) {
	const { variant = 'accent', size = 'md', className, ...rest } = props
	const classes = cn(
		'inline-flex shrink-0 cursor-pointer items-center justify-center gap-2 rounded-pill font-semibold',
		variants[variant],
		sizes[size],
		className
	)

	if (rest.href !== undefined) {
		return <AppLink {...(rest as ButtonAsLink)} className={classes} />
	}
	return <button type='button' {...(rest as ButtonAsButton)} className={classes} />
}
