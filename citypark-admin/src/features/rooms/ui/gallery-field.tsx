import { Button } from '@heroui/react'
import { ImageField } from '../../../shared/ui/image-field'
import { MAX_GALLERY, type GalleryItem, type GalleryItemErrors } from '../model/form'

type Props = {
	value: GalleryItem[]
	onChange: (value: GalleryItem[]) => void
	error?: string
	itemErrors?: Record<number, GalleryItemErrors>
}

/** The extra photos of a room: one row per photo, with buttons to move and remove it. */
export function GalleryField({ value, onChange, error, itemErrors = {} }: Props) {
	function move(index: number, to: number) {
		const next = [...value]
		const [item] = next.splice(index, 1)
		if (item !== undefined) next.splice(to, 0, item)
		onChange(next)
	}

	return (
		<div className='flex flex-col gap-6'>
			<p className='text-sm text-muted'>Дополнительные фото для страницы номера на сайте, в этом порядке</p>
			{value.map((item, index) => (
				<div key={index} className='flex flex-col gap-3'>
					<ImageField
						label={`Фото ${index + 1}`}
						aspect={3 / 2}
						withPosition={false}
						value={{ ...item, position: '' }}
						onChange={({ url, alt }) => onChange(value.map((v, i) => (i === index ? { url, alt } : v)))}
						errors={itemErrors[index]}
					/>
					<div className='flex flex-wrap gap-2'>
						<Button variant='secondary' isDisabled={index === 0} onPress={() => move(index, index - 1)}>
							Выше
						</Button>
						<Button variant='secondary' isDisabled={index === value.length - 1} onPress={() => move(index, index + 1)}>
							Ниже
						</Button>
						<Button variant='danger-soft' onPress={() => onChange(value.filter((_, i) => i !== index))}>
							Удалить
						</Button>
					</div>
				</div>
			))}
			{error !== undefined && (
				<p role='alert' className='text-sm text-danger'>
					{error}
				</p>
			)}
			<div>
				<Button
					variant='secondary'
					isDisabled={value.length >= MAX_GALLERY}
					onPress={() => onChange([...value, { url: '', alt: '' }])}
				>
					Добавить фото
				</Button>
			</div>
		</div>
	)
}
