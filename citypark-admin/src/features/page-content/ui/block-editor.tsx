import { Button, Spinner } from '@heroui/react'
import { useEffect, useState } from 'react'
import { ApiError } from '../../../shared/api'
import { formatDateTime } from '../../../shared/lib/format'
import { toastError, toastSuccess } from '../../../shared/ui/toast'
import { useSaveBlock, type ContentPage } from '../api'
import { isDirty, mapFieldErrors, toDraft, type BlockDraft } from '../model/draft'
import type { ApiBlock } from '../model/sections'
import { ImageBlock } from './image-block'
import { LinesBlock } from './lines-block'
import { RichTextBlock } from './richtext-block'
import { TextBlock } from './text-block'
import type { PhotoValue } from '../../../shared/ui/image-field'

type Props = {
	block: ApiBlock
	page: ContentPage
	/** Called when the block becomes (un)saved, so the page can guard leaving. */
	onDirtyChange: (key: string, dirty: boolean) => void
}

export function BlockEditor({ block, page, onDirtyChange }: Props) {
	const save = useSaveBlock(page)
	// The saved block is the baseline; after a save it is the server's cleaned answer.
	const [saved, setSaved] = useState(block)
	const [draft, setDraft] = useState<BlockDraft>(() => toDraft(block))
	const [errors, setErrors] = useState<Record<string, string>>({})
	const dirty = isDirty(saved, draft)

	useEffect(() => {
		onDirtyChange(block.key, dirty)
	}, [block.key, dirty, onDirtyChange])
	useEffect(() => () => onDirtyChange(block.key, false), [block.key, onDirtyChange])

	function change(next: BlockDraft) {
		setDraft(next)
		setErrors({})
	}

	function submit() {
		save.mutate(
			{ block: saved, draft },
			{
				onSuccess: (result) => {
					setSaved(result)
					setDraft(toDraft(result))
					setErrors({})
					toastSuccess('Сохранено')
				},
				onError: (error) => {
					if (error instanceof ApiError && error.status === 400) setErrors(mapFieldErrors(error.fields))
					toastError(error)
				},
			}
		)
	}

	const valueError = errors.value
	let editor
	switch (saved.type) {
		case 'text':
			editor = (
				<TextBlock
					label={saved.label}
					value={draft as string}
					maxLength={saved.maxLength}
					onChange={change}
					error={valueError}
				/>
			)
			break
		case 'lines':
			editor = <LinesBlock label={saved.label} value={draft as string[]} onChange={change} error={valueError} />
			break
		case 'richtext':
			editor = (
				<RichTextBlock
					label={saved.label}
					value={draft as string}
					maxLength={saved.maxLength}
					onChange={change}
					error={valueError}
				/>
			)
			break
		case 'image':
			editor = (
				<ImageBlock
					label={saved.label}
					value={draft as PhotoValue}
					onChange={change}
					errors={{ ...errors, ...(valueError === undefined ? {} : { url: valueError }) }}
				/>
			)
			break
	}

	return (
		<article aria-label={saved.label} className='flex flex-col gap-3 rounded-xl border border-separator bg-surface p-4'>
			<header className='flex flex-col gap-0.5'>
				<h3 className='flex items-center gap-2 text-sm font-semibold'>
					{saved.label}
					{dirty && (
						<span
							role='img'
							aria-label='Есть несохранённые изменения'
							className='inline-block size-2 rounded-full bg-accent'
						/>
					)}
				</h3>
				<code className='font-mono text-xs text-muted'>{saved.key}</code>
			</header>
			{editor}
			<footer className='flex flex-wrap items-center justify-between gap-2'>
				<Button
					variant='primary'
					size='sm'
					aria-label={`Сохранить: ${saved.label}`}
					isDisabled={!dirty}
					isPending={save.isPending}
					onPress={submit}
				>
					{({ isPending }) => (
						<>
							{isPending && <Spinner color='current' size='sm' />}
							Сохранить
						</>
					)}
				</Button>
				<span className='text-xs text-muted'>изменено {formatDateTime(saved.updatedAt)}</span>
			</footer>
		</article>
	)
}
