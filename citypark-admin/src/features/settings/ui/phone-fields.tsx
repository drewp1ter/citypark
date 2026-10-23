import { Button, Description, FieldError, Input, Label, TextField } from '@heroui/react'
import { useState } from 'react'
import { displayToNumber, type PhoneField, type PhoneName, type SettingsErrors, type SettingsForm } from '../model/form'

type Phone = SettingsForm['phones'][PhoneName]
type Props = {
	name: PhoneName
	title: string
	value: Phone
	errors: SettingsErrors
	onChange: (value: Phone) => void
}

/** The four texts of a phone and its collapsible second (landline) number. */
export function PhoneFields({ name, title, value, errors, onChange }: Props) {
	const error = (field: PhoneField) => errors[`phones.${name}.${field}`]
	const hasAlt = value.altDisplay !== '' || value.altNumber !== ''
	const [altOpen, setAltOpen] = useState(hasAlt)
	// A backend or validation error on a hidden field must not stay hidden.
	const showAlt = altOpen || error('altDisplay') !== undefined || error('altNumber') !== undefined

	function field(f: PhoneField, label: string, options: { help?: string; inputMode?: 'tel' } = {}) {
		return (
			<TextField
				name={`${name}.${f}`}
				value={value[f]}
				onChange={(text) => onChange({ ...value, [f]: text })}
				isInvalid={error(f) !== undefined}
				className='min-w-48 flex-1'
			>
				<Label>{label}</Label>
				<Input inputMode={options.inputMode} />
				{options.help !== undefined && <Description>{options.help}</Description>}
				<FieldError>{error(f)}</FieldError>
			</TextField>
		)
	}

	return (
		<fieldset className='flex flex-col gap-4'>
			<legend className='mb-2 font-medium'>{title}</legend>
			<div className='flex flex-wrap gap-4'>
				{field('label', 'Подпись')}
				{field('shortLabel', 'Короткая подпись')}
			</div>
			<div className='flex flex-wrap items-start gap-4'>
				{field('display', 'Как показывать', { help: '8 918 311-97-91' })}
				{field('number', 'Номер для звонка', { help: '+79183119791', inputMode: 'tel' })}
				<Button
					variant='secondary'
					size='sm'
					className='sm:mt-6'
					onPress={() => onChange({ ...value, number: displayToNumber(value.display) })}
				>
					Подставить номер из отображения
				</Button>
			</div>
			<div className='flex flex-col gap-4'>
				<Button
					variant='ghost'
					size='sm'
					className='self-start'
					aria-expanded={showAlt}
					onPress={() => setAltOpen(!showAlt)}
				>
					{showAlt ? '▾' : '▸'} Второй номер
				</Button>
				{showAlt && (
					<div className='flex flex-wrap items-start gap-4'>
						{field('altDisplay', 'Как показывать')}
						{field('altNumber', 'Номер для звонка', { inputMode: 'tel' })}
						<Button
							variant='secondary'
							size='sm'
							className='sm:mt-6'
							onPress={() => onChange({ ...value, altNumber: displayToNumber(value.altDisplay) })}
						>
							Подставить номер из отображения
						</Button>
					</div>
				)}
			</div>
		</fieldset>
	)
}
