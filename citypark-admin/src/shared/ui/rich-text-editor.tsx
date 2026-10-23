import Link from '@tiptap/extension-link'
import Placeholder from '@tiptap/extension-placeholder'
import { EditorContent, useEditor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import { useEffect, useId, useRef } from 'react'
import { normalizeHtml, textLength } from '../lib/rich-text'
import { RichTextToolbar, type Placeholder as PlaceholderItem } from './rich-text-toolbar'

export type RichTextProfile = 'article' | 'short'

type Props = {
	label: string
	value: string // HTML
	onChange: (html: string) => void
	profile: RichTextProfile
	placeholders?: readonly PlaceholderItem[] // e.g. { name: 'smokingFine', label: 'Штраф за курение' }
	maxLength?: number // visible characters; shows a counter «123 / 1000», red when over
	error?: string
}

/** Same element styles as the site's RichText: paragraph spacing, list markers, headings below the page h1. */
const CONTENT_CLASS = [
	'min-h-32 px-3 py-2 text-sm outline-none',
	'[&_p]:my-2 [&_ul]:my-2 [&_ul]:list-disc [&_ul]:pl-6 [&_ol]:my-2 [&_ol]:list-decimal [&_ol]:pl-6',
	'[&_h2]:mt-4 [&_h2]:mb-2 [&_h2]:text-xl [&_h2]:font-semibold',
	'[&_h3]:mt-3 [&_h3]:mb-2 [&_h3]:text-lg [&_h3]:font-semibold',
	'[&_blockquote]:my-2 [&_blockquote]:border-l-4 [&_blockquote]:border-separator [&_blockquote]:pl-4 [&_blockquote]:text-muted',
	'[&_a]:text-accent-soft-foreground [&_a]:underline',
	'[&_.is-editor-empty:first-child::before]:pointer-events-none [&_.is-editor-empty:first-child::before]:float-left [&_.is-editor-empty:first-child::before]:h-0 [&_.is-editor-empty:first-child::before]:text-muted [&_.is-editor-empty:first-child::before]:content-[attr(data-placeholder)]',
].join(' ')

export function RichTextEditor({ label, value, onChange, profile, placeholders, maxLength, error }: Props) {
	const id = useId()
	// The editor is created once; callbacks read the newest props through a ref.
	const onChangeRef = useRef(onChange)
	useEffect(() => {
		onChangeRef.current = onChange
	})

	const editor = useEditor({
		extensions: [
			StarterKit.configure({
				code: false,
				codeBlock: false,
				strike: false,
				underline: false,
				horizontalRule: false,
				link: false, // configured below
				heading: profile === 'article' ? { levels: [2, 3] } : false,
				blockquote: profile === 'article' ? {} : false,
			}),
			Link.configure({
				openOnClick: false,
				protocols: ['http', 'https', 'mailto', 'tel'],
				HTMLAttributes: { target: null, rel: null }, // the backend adds target and rel
			}),
			Placeholder.configure({ placeholder: 'Введите текст' }),
		],
		content: value,
		editorProps: {
			attributes: {
				role: 'textbox',
				'aria-multiline': 'true',
				'aria-label': label,
				class: CONTENT_CLASS,
			},
		},
		onUpdate: ({ editor: e }) => onChangeRef.current(normalizeHtml(e.getHTML())),
	})

	// Follow value changes that did not come from typing (loading a record, resetting a form).
	useEffect(() => {
		if (editor === null) return
		if (normalizeHtml(editor.getHTML()) !== value) {
			editor.commands.setContent(value, { emitUpdate: false })
		}
	}, [editor, value])

	const length = textLength(value)
	const over = maxLength !== undefined && length > maxLength

	return (
		<div className='flex flex-col gap-1.5'>
			<span id={`${id}-label`} className='text-sm font-medium'>
				{label}
			</span>
			<div
				className={`rounded-lg border bg-field focus-within:outline-2 focus-within:outline-offset-2 focus-within:outline-focus ${
					error !== undefined || over ? 'border-danger' : 'border-separator'
				}`}
			>
				{editor !== null && <RichTextToolbar editor={editor} profile={profile} placeholders={placeholders} />}
				<EditorContent editor={editor} />
			</div>
			<div className='flex flex-wrap justify-between gap-2 text-xs'>
				<div className='flex flex-col gap-1'>
					{placeholders !== undefined && placeholders.length > 0 && (
						<p className='text-muted'>Цены: {placeholders.map((p) => `{{${p.name}}} — ${p.label}`).join('; ')}</p>
					)}
					{error !== undefined && (
						<p role='alert' className='text-danger'>
							{error}
						</p>
					)}
				</div>
				{maxLength !== undefined && (
					<span className={over ? 'text-danger' : 'text-muted'} data-testid={`${label}-counter`}>
						{length} / {maxLength}
					</span>
				)}
			</div>
		</div>
	)
}
