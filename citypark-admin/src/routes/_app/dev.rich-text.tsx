import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { PageHeader } from '../../shared/ui/page-header'
import { RichTextEditor } from '../../shared/ui/rich-text-editor'

const PLACEHOLDERS = [
	{ name: 'smokingFine', label: 'Штраф за курение' },
	{ name: 'minRoomPrice', label: 'Минимальная цена номера' },
] as const

/** Hidden demo page (not in the navigation): the e2e test of the RichTextEditor uses it. */
function DevRichText() {
	const [short, setShort] = useState('')
	const [article, setArticle] = useState('')
	return (
		<>
			<PageHeader title='Демо: редактор текста' />
			<div className='flex max-w-2xl flex-col gap-8'>
				<div>
					<RichTextEditor
						label='Короткий текст'
						profile='short'
						value={short}
						onChange={setShort}
						placeholders={PLACEHOLDERS}
						maxLength={100}
					/>
					<pre data-testid='short-html' className='mt-2 overflow-x-auto text-xs whitespace-pre-wrap'>
						{short}
					</pre>
				</div>
				<div>
					<RichTextEditor label='Текст статьи' profile='article' value={article} onChange={setArticle} />
					<pre data-testid='article-html' className='mt-2 overflow-x-auto text-xs whitespace-pre-wrap'>
						{article}
					</pre>
				</div>
			</div>
		</>
	)
}

export const Route = createFileRoute('/_app/dev/rich-text')({
	component: DevRichText,
})
