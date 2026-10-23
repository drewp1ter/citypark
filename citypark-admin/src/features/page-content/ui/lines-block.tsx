import { Button, Input } from '@heroui/react'
import { MAX_LINES } from '../model/draft'

type Props = { label: string; value: string[]; onChange: (v: string[]) => void; error?: string }

export function LinesBlock({ label, value, onChange, error }: Props) {
	function setLine(index: number, text: string) {
		onChange(value.map((line, i) => (i === index ? text : line)))
	}

	function move(index: number, by: -1 | 1) {
		const a = value[index]
		const b = value[index + by]
		if (a === undefined || b === undefined) return
		onChange(value.map((line, i) => (i === index ? b : i === index + by ? a : line)))
	}

	const count = value.filter((line) => line.trim() !== '').length

	return (
		<div role='group' aria-label={label} className='flex flex-col gap-2'>
			<ul className='flex flex-col gap-2'>
				{value.map((line, index) => (
					// Lines have no identity of their own; the index is the only stable handle while editing.
					<li key={index} className='flex flex-wrap items-center gap-2'>
						<Input
							aria-label={`${label}: строка ${index + 1}`}
							className='min-w-48 flex-1'
							value={line}
							onChange={(e) => setLine(index, e.target.value)}
						/>
						<Button
							size='sm'
							isIconOnly
							variant='ghost'
							aria-label={`Строка ${index + 1} выше`}
							isDisabled={index === 0}
							onPress={() => move(index, -1)}
						>
							↑
						</Button>
						<Button
							size='sm'
							isIconOnly
							variant='ghost'
							aria-label={`Строка ${index + 1} ниже`}
							isDisabled={index === value.length - 1}
							onPress={() => move(index, 1)}
						>
							↓
						</Button>
						<Button
							size='sm'
							variant='danger-soft'
							aria-label={`Удалить строку ${index + 1}`}
							onPress={() => onChange(value.filter((_, i) => i !== index))}
						>
							Удалить строку
						</Button>
					</li>
				))}
			</ul>
			<div className='flex flex-wrap items-center justify-between gap-2'>
				<Button
					size='sm'
					variant='secondary'
					isDisabled={value.length >= MAX_LINES}
					onPress={() => onChange([...value, ''])}
				>
					Добавить строку
				</Button>
				<span className={`text-xs ${count < 1 || value.length > MAX_LINES ? 'text-danger' : 'text-muted'}`}>
					Строк: {count} (1–{MAX_LINES})
				</span>
			</div>
			{error !== undefined && (
				<p role='alert' className='text-sm text-danger'>
					{error}
				</p>
			)}
		</div>
	)
}
