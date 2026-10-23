import { Button } from '@heroui/react'
import type { Editor } from '@tiptap/react'
import { useEditorState } from '@tiptap/react'
import { useState, type ReactNode } from 'react'
import type { RichTextProfile } from './rich-text-editor'

export type Placeholder = { name: string; label: string }

type Props = {
	editor: Editor
	profile: RichTextProfile
	placeholders?: readonly Placeholder[]
}

type ToolButtonProps = { label: string; active?: boolean; disabled?: boolean; onPress: () => void; children: ReactNode }

function ToolButton({ label, active, disabled, onPress, children }: ToolButtonProps) {
	return (
		<Button
			size='sm'
			isIconOnly
			variant={active === true ? 'primary' : 'ghost'}
			aria-label={label}
			aria-pressed={active ?? false}
			isDisabled={disabled}
			onPress={onPress}
		>
			{children}
		</Button>
	)
}

export function RichTextToolbar({ editor, profile, placeholders }: Props) {
	const [linkOpen, setLinkOpen] = useState(false)
	const [href, setHref] = useState('')
	const [priceOpen, setPriceOpen] = useState(false)

	const state = useEditorState({
		editor,
		selector: ({ editor: e }) => ({
			bold: e.isActive('bold'),
			italic: e.isActive('italic'),
			bullet: e.isActive('bulletList'),
			ordered: e.isActive('orderedList'),
			link: e.isActive('link'),
			h2: e.isActive('heading', { level: 2 }),
			h3: e.isActive('heading', { level: 3 }),
			quote: e.isActive('blockquote'),
			canUndo: e.can().undo(),
			canRedo: e.can().redo(),
		}),
	})

	function openLink() {
		setHref((editor.getAttributes('link').href as string | undefined) ?? '')
		setLinkOpen((open) => !open)
		setPriceOpen(false)
	}

	function applyLink() {
		const value = href.trim()
		if (value === '') {
			editor.chain().focus().extendMarkRange('link').unsetLink().run()
		} else {
			editor.chain().focus().extendMarkRange('link').setLink({ href: value }).run()
		}
		setLinkOpen(false)
	}

	function removeLink() {
		editor.chain().focus().extendMarkRange('link').unsetLink().run()
		setLinkOpen(false)
	}

	function insertPlaceholder(name: string) {
		editor.chain().focus().insertContent(`{{${name}}}`).run()
		setPriceOpen(false)
	}

	return (
		<div className='flex flex-col gap-2 border-b border-separator p-1'>
			<div role='toolbar' aria-label='Форматирование' className='flex flex-wrap items-center gap-1'>
				<ToolButton label='Жирный' active={state.bold} onPress={() => editor.chain().focus().toggleBold().run()}>
					<b>Ж</b>
				</ToolButton>
				<ToolButton label='Курсив' active={state.italic} onPress={() => editor.chain().focus().toggleItalic().run()}>
					<i>К</i>
				</ToolButton>
				{profile === 'article' && (
					<>
						<ToolButton
							label='Заголовок 2'
							active={state.h2}
							onPress={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
						>
							H2
						</ToolButton>
						<ToolButton
							label='Заголовок 3'
							active={state.h3}
							onPress={() => editor.chain().focus().toggleHeading({ level: 3 }).run()}
						>
							H3
						</ToolButton>
						<ToolButton
							label='Цитата'
							active={state.quote}
							onPress={() => editor.chain().focus().toggleBlockquote().run()}
						>
							«»
						</ToolButton>
					</>
				)}
				<ToolButton
					label='Маркированный список'
					active={state.bullet}
					onPress={() => editor.chain().focus().toggleBulletList().run()}
				>
					•
				</ToolButton>
				<ToolButton
					label='Нумерованный список'
					active={state.ordered}
					onPress={() => editor.chain().focus().toggleOrderedList().run()}
				>
					1.
				</ToolButton>
				<ToolButton label='Ссылка' active={state.link || linkOpen} onPress={openLink}>
					🔗
				</ToolButton>
				<ToolButton label='Отменить' disabled={!state.canUndo} onPress={() => editor.chain().focus().undo().run()}>
					↶
				</ToolButton>
				<ToolButton label='Повторить' disabled={!state.canRedo} onPress={() => editor.chain().focus().redo().run()}>
					↷
				</ToolButton>
				{placeholders !== undefined && placeholders.length > 0 && (
					<Button
						size='sm'
						variant='secondary'
						aria-expanded={priceOpen}
						onPress={() => {
							setPriceOpen((open) => !open)
							setLinkOpen(false)
						}}
					>
						Вставить цену
					</Button>
				)}
			</div>
			{linkOpen && (
				<form
					className='flex flex-wrap items-end gap-2 px-1 pb-1'
					onSubmit={(e) => {
						e.preventDefault()
						applyLink()
					}}
				>
					<label className='flex min-w-48 flex-1 flex-col gap-1 text-sm'>
						Адрес ссылки
						<input
							type='text'
							value={href}
							placeholder='https://… или /rooms'
							onChange={(e) => setHref(e.target.value)}
							className='rounded-lg border border-separator bg-field px-3 py-2 text-sm outline-none focus-visible:outline-2 focus-visible:outline-focus'
						/>
					</label>
					<Button size='sm' type='submit'>
						Применить
					</Button>
					<Button size='sm' variant='secondary' isDisabled={!state.link} onPress={removeLink}>
						Убрать ссылку
					</Button>
				</form>
			)}
			{priceOpen && placeholders !== undefined && (
				<ul aria-label='Цены для вставки' className='flex flex-wrap gap-2 px-1 pb-1'>
					{placeholders.map((p) => (
						<li key={p.name}>
							<Button size='sm' variant='outline' onPress={() => insertPlaceholder(p.name)}>
								{p.label}
							</Button>
						</li>
					))}
				</ul>
			)}
		</div>
	)
}
