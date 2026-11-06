'use client'

import { useState, type ComponentProps } from 'react'
import { formatPhoneInput } from '@/shared/lib'
import { TextField } from './fields'

type PhoneFieldProps = Omit<ComponentProps<typeof TextField>, 'type' | 'value' | 'onChange' | 'defaultValue'> & {
	value?: string
	onValueChange?: (value: string) => void
	defaultValue?: string
}

/** Phone input with the "+7 ___ ___-__-__" mask. The submitted value is the formatted text. */
export function PhoneField({ defaultValue = '', value: controlled, onValueChange, ...props }: PhoneFieldProps) {
	const [value, setValue] = useState(formatPhoneInput(defaultValue))

	return (
		<TextField
			{...props}
			type='tel'
			inputMode='tel'
			autoComplete='tel'
			placeholder='+7 ___ ___-__-__'
			value={controlled === undefined ? value : formatPhoneInput(controlled)}
			onChange={(event) => {
				const next = formatPhoneInput(event.target.value)
				setValue(next)
				onValueChange?.(next)
			}}
		/>
	)
}
