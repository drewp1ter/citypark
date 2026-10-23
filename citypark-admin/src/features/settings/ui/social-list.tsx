import { Button, FieldError, Input, Label, TextField } from '@heroui/react'
import { MAX_SOCIAL, type SettingsErrors, type SettingsForm } from '../model/form'

type Social = SettingsForm['social']
type Props = { value: Social; errors: SettingsErrors; onChange: (value: Social) => void }

const EMPTY = { name: '', label: '', url: 'https://', display: '' }

/** Rows of social links: edit, add (up to MAX_SOCIAL), remove and reorder. */
export function SocialList({ value, errors, onChange }: Props) {
	function setRow(index: number, patch: Partial<Social[number]>) {
		onChange(value.map((row, i) => (i === index ? { ...row, ...patch } : row)))
	}

	function move(index: number, by: -1 | 1) {
		const a = value[index]
		const b = value[index + by]
		if (a === undefined || b === undefined) return
		onChange(value.map((row, i) => (i === index ? b : i === index + by ? a : row)))
	}

	function field(index: number, f: keyof Social[number], label: string) {
		const error = errors[`social[${index}].${f}`]
		return (
			<TextField
				name={`social.${index}.${f}`}
				value={value[index]?.[f] ?? ''}
				onChange={(text) => setRow(index, { [f]: text })}
				isInvalid={error !== undefined}
				className='min-w-48 flex-1'
			>
				<Label>{label}</Label>
				<Input inputMode={f === 'url' ? 'url' : undefined} />
				<FieldError>{error}</FieldError>
			</TextField>
		)
	}

	return (
		<div className='flex flex-col gap-4'>
			<ul className='flex flex-col gap-4'>
				{value.map((_, index) => (
					// Rows have no identity of their own; the index is the only stable handle while editing.
					<li key={index}>
						<fieldset className='flex flex-col gap-3 rounded-xl border border-separator p-3'>
							<legend className='px-1 text-sm text-muted'>Соцсеть {index + 1}</legend>
							<div className='flex flex-wrap gap-4'>
								{field(index, 'name', 'Название')}
								{field(index, 'label', 'Короткая подпись')}
							</div>
							<div className='flex flex-wrap gap-4'>
								{field(index, 'url', 'Ссылка (https://…)')}
								{field(index, 'display', 'Как показывать')}
							</div>
							<div className='flex flex-wrap items-center gap-2'>
								<Button
									size='sm'
									isIconOnly
									variant='ghost'
									aria-label={`Соцсеть ${index + 1} выше`}
									isDisabled={index === 0}
									onPress={() => move(index, -1)}
								>
									↑
								</Button>
								<Button
									size='sm'
									isIconOnly
									variant='ghost'
									aria-label={`Соцсеть ${index + 1} ниже`}
									isDisabled={index === value.length - 1}
									onPress={() => move(index, 1)}
								>
									↓
								</Button>
								<Button
									size='sm'
									variant='danger-soft'
									aria-label={`Удалить соцсеть ${index + 1}`}
									onPress={() => onChange(value.filter((_, i) => i !== index))}
								>
									Удалить
								</Button>
							</div>
						</fieldset>
					</li>
				))}
			</ul>
			<div className='flex flex-wrap items-center justify-between gap-2'>
				<Button
					size='sm'
					variant='secondary'
					isDisabled={value.length >= MAX_SOCIAL}
					onPress={() => onChange([...value, { ...EMPTY }])}
				>
					Добавить соцсеть
				</Button>
				<span className='text-xs text-muted'>
					Соцсетей: {value.length} (не больше {MAX_SOCIAL})
				</span>
			</div>
			{errors.social !== undefined && (
				<p role='alert' className='text-sm text-danger'>
					{errors.social}
				</p>
			)}
		</div>
	)
}
