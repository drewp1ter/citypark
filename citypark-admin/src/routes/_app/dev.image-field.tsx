import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { ImageField, type PhotoValue } from '../../shared/ui/image-field'
import { PageHeader } from '../../shared/ui/page-header'

/** Hidden demo page (not in the navigation): the e2e test of the ImageField uses it. */
function DevImageField() {
	const [value, setValue] = useState<PhotoValue>({ url: '', alt: '', position: '' })
	return (
		<>
			<PageHeader title='Демо: поле с фото' />
			<div className='max-w-xl'>
				<ImageField label='Фото' value={value} onChange={setValue} allowEmpty />
				<pre data-testid='value' className='mt-4 overflow-x-auto text-xs'>
					{JSON.stringify(value)}
				</pre>
			</div>
		</>
	)
}

export const Route = createFileRoute('/_app/dev/image-field')({
	component: DevImageField,
})
