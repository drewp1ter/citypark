import { Button, FieldError, Input, Label, TextField } from '@heroui/react'
import { useRef, useState } from 'react'
import { insertPlaceholder, PRICE_PLACEHOLDERS } from '../model/placeholders'

type Props = { label: string; value: string; maxLength: number; onChange: (v: string) => void; error?: string }

export function TextBlock({ label, value, maxLength, onChange, error }: Props) {
	const inputRef = useRef<HTMLInputElement>(null)
	const [menuOpen, setMenuOpen] = useState(false)
	const over = value.length > maxLength

	function insert(name: string) {
		const input = inputRef.current
		const start = input?.selectionStart ?? value.length
		const end = input?.selectionEnd ?? value.length
		const next = insertPlaceholder(value, name, start, end)
		onChange(next.value)
		setMenuOpen(false)
		requestAnimationFrame(() => {
			input?.focus()
			input?.setSelectionRange(next.caret, next.caret)
		})
	}

	return (
		<div className='flex flex-col gap-2'>
			<TextField name='value' value={value} onChange={onChange} isInvalid={error !== undefined || over}>
				<Label>{label}</Label>
				<Input ref={inputRef} />
				<FieldError>{error}</FieldError>
			</TextField>
			<div className='flex flex-wrap items-center justify-between gap-2'>
				<Button size='sm' variant='secondary' aria-expanded={menuOpen} onPress={() => setMenuOpen((open) => !open)}>
					Вставить цену
				</Button>
				<span className={`text-xs ${over ? 'text-danger' : 'text-muted'}`}>
					{value.length} / {maxLength}
				</span>
			</div>
			{menuOpen && (
				<ul aria-label='Цены для вставки' className='flex flex-wrap gap-2'>
					{PRICE_PLACEHOLDERS.map((p) => (
						<li key={p.name}>
							<Button size='sm' variant='outline' onPress={() => insert(p.name)}>
								{p.label}
							</Button>
						</li>
					))}
				</ul>
			)}
		</div>
	)
}
