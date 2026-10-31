import { Input, Label, TextField } from '@heroui/react'
import { useState } from 'react'

export function TargetPicker({
	label,
	items,
	selected,
	onChange,
	error,
}: {
	label: string
	items: { id: number; name: string }[]
	selected: number[]
	onChange: (ids: number[]) => void
	error?: string
}) {
	const [search, setSearch] = useState('')
	return (
		<fieldset className='min-w-0 rounded-xl border border-border p-3'>
			<legend>
				{label} · выбрано {selected.length}
			</legend>
			<TextField value={search} onChange={setSearch} className='mb-2'>
				<Label>Поиск: {label.toLowerCase()}</Label>
				<Input type='search' />
			</TextField>
			<div className='max-h-64 overflow-y-auto'>
				{items
					.filter((item) => item.name.toLocaleLowerCase('ru').includes(search.trim().toLocaleLowerCase('ru')))
					.map((item) => (
						<label key={item.id} className='flex min-h-11 items-center gap-2 py-2 wrap-anywhere'>
							<input
								type='checkbox'
								checked={selected.includes(item.id)}
								onChange={(e) =>
									onChange(e.target.checked ? [...selected, item.id] : selected.filter((id) => id !== item.id))
								}
							/>
							{item.name}
						</label>
					))}
			</div>
			{error && (
				<p role='alert' className='text-sm text-danger'>
					{error}
				</p>
			)}
		</fieldset>
	)
}
