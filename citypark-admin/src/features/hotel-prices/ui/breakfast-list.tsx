import { Button, Checkbox, FieldError, Input, Label, TextField } from '@heroui/react'
import { EMPTY_ITEM, MAX_BREAKFAST, moveItem, type BreakfastItemForm, type HotelPricesErrors } from '../model/form'

type Props = { value: BreakfastItemForm[]; errors: HotelPricesErrors; onChange: (value: BreakfastItemForm[]) => void }

/** Rows of breakfast dishes: edit, add (up to MAX_BREAKFAST), remove and reorder. */
export function BreakfastList({ value, errors, onChange }: Props) {
	function setRow(index: number, patch: Partial<BreakfastItemForm>) {
		onChange(value.map((row, i) => (i === index ? { ...row, ...patch } : row)))
	}

	function field(index: number, f: 'name' | 'shortName' | 'price', label: string) {
		const error = errors[`breakfast[${index}].${f}`]
		return (
			<TextField
				name={`breakfast.${index}.${f}`}
				value={value[index]?.[f] ?? ''}
				onChange={(text) => setRow(index, { [f]: text })}
				isInvalid={error !== undefined}
				className='min-w-48 flex-1'
			>
				<Label>{label}</Label>
				<Input inputMode={f === 'price' ? 'decimal' : undefined} />
				<FieldError>{error}</FieldError>
			</TextField>
		)
	}

	return (
		<div className='flex flex-col gap-4'>
			<ul className='flex flex-col gap-4'>
				{value.map((item, index) => (
					// Rows have no identity of their own; the index is the only stable handle while editing.
					<li key={index}>
						<fieldset className='flex flex-col gap-3 rounded-xl border border-separator p-3'>
							<legend className='px-1 text-sm text-muted'>Блюдо {index + 1}</legend>
							{field(index, 'name', 'Название')}
							<div className='flex flex-wrap gap-4'>
								{field(index, 'shortName', 'Короткое название для телефона')}
								{field(index, 'price', 'Цена, ₽')}
							</div>
							<Checkbox
								isSelected={item.desktopOnly}
								onChange={(desktopOnly) => setRow(index, { desktopOnly })}
								name={`breakfast.${index}.desktopOnly`}
							>
								<Checkbox.Control>
									<Checkbox.Indicator />
								</Checkbox.Control>
								<Checkbox.Content>
									<Label>Только на компьютере</Label>
								</Checkbox.Content>
							</Checkbox>
							<div className='flex flex-wrap items-center gap-2'>
								<Button
									size='sm'
									isIconOnly
									variant='ghost'
									aria-label={`Блюдо ${index + 1} выше`}
									isDisabled={index === 0}
									onPress={() => onChange(moveItem(value, index, -1))}
								>
									↑
								</Button>
								<Button
									size='sm'
									isIconOnly
									variant='ghost'
									aria-label={`Блюдо ${index + 1} ниже`}
									isDisabled={index === value.length - 1}
									onPress={() => onChange(moveItem(value, index, 1))}
								>
									↓
								</Button>
								<Button
									size='sm'
									variant='danger-soft'
									aria-label={`Удалить блюдо ${index + 1}`}
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
					isDisabled={value.length >= MAX_BREAKFAST}
					onPress={() => onChange([...value, { ...EMPTY_ITEM }])}
				>
					Добавить блюдо
				</Button>
				<span className='text-xs text-muted'>
					Блюд: {value.length} (не больше {MAX_BREAKFAST})
				</span>
			</div>
			{errors.breakfast !== undefined && (
				<p role='alert' className='text-sm text-danger'>
					{errors.breakfast}
				</p>
			)}
		</div>
	)
}
