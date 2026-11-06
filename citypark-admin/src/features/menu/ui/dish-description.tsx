import { Button } from '@heroui/react'
import { useState, type FormEvent } from 'react'
import { ApiError } from '../../../shared/api'
import { RichTextEditor } from '../../../shared/ui/rich-text-editor'
import { toastError } from '../../../shared/ui/toast'
import { useSaveProductDescription } from '../api'
import type { ApiProduct } from '../model/group'

export function DishDescription({ dish, onClose }: { dish: ApiProduct; onClose: () => void }) {
	const [value, setValue] = useState(dish.fullDescription ?? '')
	const save = useSaveProductDescription(dish.id)
	function submit(event: FormEvent<HTMLFormElement>) {
		event.preventDefault()
		if (save.isPending) return
		save.mutate(value, {
			onSuccess: (saved, submitted) => setValue((current) => (current === submitted ? saved.fullDescription : current)),
			onError: toastError,
		})
	}
	return (
		<section className='rounded-xl border border-border bg-surface p-4' aria-label={`Описание: ${dish.name}`}>
			<h2 className='mb-3 text-lg font-semibold'>{dish.name}</h2>
			<p className='mb-3 text-sm text-muted'>
				Полное описание показывается только на странице блюда. Если оставить его пустым, будет показано описание из
				iiko.
			</p>
			<form onSubmit={submit} className='flex flex-col gap-3'>
				<RichTextEditor
					label='Полное описание'
					value={value}
					onChange={setValue}
					profile='short'
					maxLength={10000}
					error={save.error instanceof ApiError ? save.error.fields.fullDescription : undefined}
				/>
				<div className='flex flex-wrap gap-2'>
					<Button type='submit' isPending={save.isPending}>
						Сохранить описание
					</Button>
					<Button type='button' variant='secondary' onPress={onClose} isDisabled={save.isPending}>
						Закрыть
					</Button>
				</div>
			</form>
		</section>
	)
}
