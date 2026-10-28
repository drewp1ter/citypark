import { useId, type ComponentPropsWithoutRef, type ReactNode } from 'react'
import { cn } from '@/shared/lib'
import { ClearableInput } from './clearable-input'

/** Look of every text input, select and textarea (the design fills them with the page colour). */
export const controlClass =
	'min-h-12 w-full rounded-xs border border-border bg-bg px-4 py-3 font-normal text-text placeholder:text-text-muted aria-[invalid=true]:border-danger'

/** Label + spacing shared by every field, so `Select` (React Aria) matches `FieldShell`. */
export const fieldLabelClass = 'flex flex-col gap-1.5 text-[13px] font-semibold md:text-sm'

export const fieldErrorClass = 'text-danger text-sm font-normal'

type FieldShellProps = {
	label: ReactNode
	error?: string
	errorId: string
	className?: string
	children: ReactNode
}

function FieldShell({ label, error, errorId, className, children }: FieldShellProps) {
	return (
		<label className={cn(fieldLabelClass, className)}>
			{label}
			{children}
			{error && (
				<span id={errorId} role='alert' className={fieldErrorClass}>
					{error}
				</span>
			)}
		</label>
	)
}

const invalid = (error: string | undefined, errorId: string) => ({
	'aria-invalid': error ? (true as const) : undefined,
	'aria-describedby': error ? errorId : undefined,
})

type CommonProps = {
	/** Text or markup: use two spans (`md:hidden` / `hidden md:inline`) for a shorter label on mobile. */
	label: ReactNode
	error?: string
	className?: string
}

type TextFieldProps = Omit<ComponentPropsWithoutRef<'input'>, 'className'> &
	CommonProps & {
		/** Shows a «×» button while the input has text. */
		clearable?: boolean
		onClear?: () => void
	}

/** Label + input + error message. Works for text, tel, date, time, email … */
export function TextField({ label, error, className, clearable, onClear, ...props }: TextFieldProps) {
	const errorId = useId()
	return (
		<FieldShell label={label} error={error} errorId={errorId} className={className}>
			{clearable ? (
				<ClearableInput {...props} {...invalid(error, errorId)} onClear={onClear} className={controlClass} />
			) : (
				<input {...props} {...invalid(error, errorId)} className={controlClass} />
			)}
		</FieldShell>
	)
}

type TextareaFieldProps = Omit<ComponentPropsWithoutRef<'textarea'>, 'className'> & CommonProps

export function TextareaField({ label, error, className, ...props }: TextareaFieldProps) {
	const errorId = useId()
	return (
		<FieldShell label={label} error={error} errorId={errorId} className={className}>
			<textarea {...props} {...invalid(error, errorId)} className={cn(controlClass, 'resize-y')} />
		</FieldShell>
	)
}

type CheckboxFieldProps = Omit<ComponentPropsWithoutRef<'input'>, 'type' | 'className'> & {
	/** The text next to the box; may contain a link. */
	children: ReactNode
	error?: string
	className?: string
}

export function CheckboxField({ children, error, className, ...props }: CheckboxFieldProps) {
	const errorId = useId()
	return (
		<div className={className}>
			<label className='flex min-h-11 cursor-pointer items-start gap-2.5 py-2.5 text-sm'>
				<input
					type='checkbox'
					{...props}
					{...invalid(error, errorId)}
					className='accent-accent mt-0.5 size-5 shrink-0 cursor-pointer md:size-[18px]'
				/>
				<span>{children}</span>
			</label>
			{error && (
				<p id={errorId} role='alert' className='text-danger mt-1.5 text-sm'>
					{error}
				</p>
			)}
		</div>
	)
}
