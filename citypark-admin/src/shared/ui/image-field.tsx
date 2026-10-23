import { Button, Description, FieldError, Input, Label, Spinner, TextField } from '@heroui/react'
import {
	useEffect,
	useId,
	useRef,
	useState,
	type ChangeEvent,
	type DragEvent,
	type KeyboardEvent,
	type MouseEvent,
} from 'react'
import { apiUpload } from '../api'
import { parsePosition, toPosition } from '../lib/focal-point'
import { imageSrc } from '../lib/image-src'

export type PhotoValue = { url: string; alt: string; position: string }

type Props = {
	label: string
	value: PhotoValue
	onChange: (v: PhotoValue) => void
	/** Aspect ratio of the preview, like the place the photo is shown on the site (e.g. 4 / 3). */
	aspect?: number
	allowEmpty?: boolean // shows «Убрать фото» (news); required fields do not
	errors?: Partial<Record<'url' | 'alt' | 'position', string>>
}

const MAX_BYTES = 10 * 1024 * 1024
const ACCEPT = 'image/jpeg,image/png,image/webp'
const ARROW_STEP = 5

function message(err: unknown): string {
	return err instanceof Error ? err.message : 'Не удалось загрузить фото'
}

export function ImageField({ label, value, onChange, aspect = 4 / 3, allowEmpty = false, errors = {} }: Props) {
	const inputId = useId()
	const inputRef = useRef<HTMLInputElement>(null)
	const [uploading, setUploading] = useState(false)
	const [uploadError, setUploadError] = useState<string>()
	const [dragging, setDragging] = useState(false)
	// The upload finishes later than the render that started it: always merge into the newest value.
	const latest = useRef(value)
	useEffect(() => {
		latest.current = value
	})

	const position = value.position === '' ? '50% 50%' : value.position
	const point = parsePosition(value.position)
	const hasPhoto = value.url !== ''

	async function upload(file: File | undefined) {
		if (file === undefined) return
		if (file.size > MAX_BYTES) {
			setUploadError('Файл больше 10 МБ')
			return
		}
		setUploadError(undefined)
		setUploading(true)
		try {
			const { url } = await apiUpload('/uploads', file)
			onChange({ ...latest.current, url, position: '' })
		} catch (err) {
			setUploadError(message(err))
		} finally {
			setUploading(false)
		}
	}

	function onFileChange(e: ChangeEvent<HTMLInputElement>) {
		void upload(e.target.files?.[0])
		e.target.value = '' // allows choosing the same file again
	}

	function onDrop(e: DragEvent) {
		e.preventDefault()
		setDragging(false)
		void upload(e.dataTransfer.files[0])
	}

	function setPoint(x: number, y: number) {
		onChange({ ...value, position: toPosition(x, y) })
	}

	function onPreviewClick(e: MouseEvent<HTMLDivElement>) {
		const box = e.currentTarget.getBoundingClientRect()
		if (box.width === 0 || box.height === 0) return
		setPoint((e.clientX - box.left) / box.width, (e.clientY - box.top) / box.height)
	}

	function onPreviewKeyDown(e: KeyboardEvent<HTMLDivElement>) {
		const move: Record<string, [number, number]> = {
			ArrowLeft: [-ARROW_STEP, 0],
			ArrowRight: [ARROW_STEP, 0],
			ArrowUp: [0, -ARROW_STEP],
			ArrowDown: [0, ARROW_STEP],
		}
		const delta = move[e.key]
		if (delta === undefined) return
		e.preventDefault()
		setPoint((point.x + delta[0]) / 100, (point.y + delta[1]) / 100)
	}

	const errorText = uploadError ?? errors.url

	return (
		<div className='flex flex-col gap-3'>
			<span className='text-sm font-medium'>{label}</span>
			<input
				ref={inputRef}
				id={inputId}
				type='file'
				accept={ACCEPT}
				className='peer sr-only'
				aria-label={hasPhoto ? `${label}: выбрать файл` : undefined}
				onChange={onFileChange}
			/>
			{hasPhoto ? (
				<div className='flex flex-wrap items-start gap-4'>
					<div
						role='group'
						tabIndex={0}
						aria-label={`${label}: фокусная точка. Нажмите на фото или используйте стрелки`}
						aria-describedby={`${inputId}-position`}
						className='relative w-full max-w-md cursor-crosshair overflow-hidden rounded-lg border border-separator focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus'
						style={{ aspectRatio: aspect }}
						onClick={onPreviewClick}
						onKeyDown={onPreviewKeyDown}
					>
						<img
							src={imageSrc(value.url)}
							alt=''
							draggable={false}
							className='size-full object-cover'
							style={{ objectPosition: position }}
						/>
						<span
							aria-hidden='true'
							data-testid='focal-marker'
							className='pointer-events-none absolute size-5 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-white bg-accent shadow-md'
							style={{ left: `${point.x}%`, top: `${point.y}%` }}
						/>
						{uploading && (
							<div className='absolute inset-0 flex items-center justify-center bg-black/40'>
								<Spinner aria-label='Загрузка фото' />
							</div>
						)}
					</div>
					<div className='flex flex-col gap-1'>
						<img
							src={imageSrc(value.url)}
							alt=''
							className='size-24 rounded-lg border border-separator object-cover'
							style={{ objectPosition: position }}
						/>
						<span className='text-xs text-muted'>Квадратная обрезка</span>
					</div>
				</div>
			) : (
				<label
					htmlFor={inputId}
					className={`flex min-h-32 cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border-2 border-dashed p-4 text-center peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-focus ${
						dragging ? 'border-accent bg-accent/10' : 'border-separator'
					}`}
					onDragOver={(e) => {
						e.preventDefault()
						setDragging(true)
					}}
					onDragLeave={() => setDragging(false)}
					onDrop={onDrop}
				>
					{uploading ? <Spinner aria-label='Загрузка фото' /> : <span>Перетащите фото или выберите файл</span>}
					<span className='text-xs text-muted'>JPEG, PNG или WebP, до 10 МБ</span>
				</label>
			)}
			{hasPhoto && (
				<>
					<p id={`${inputId}-position`} className='text-xs text-muted'>
						Фокусная точка: {position}
					</p>
					<div className='flex flex-wrap gap-2'>
						<Button variant='secondary' isDisabled={uploading} onPress={() => inputRef.current?.click()}>
							Заменить
						</Button>
						<Button
							variant='secondary'
							isDisabled={value.position === ''}
							onPress={() => onChange({ ...value, position: '' })}
						>
							Сбросить точку
						</Button>
						{allowEmpty && (
							<Button variant='danger-soft' onPress={() => onChange({ url: '', alt: '', position: '' })}>
								Убрать фото
							</Button>
						)}
					</div>
				</>
			)}
			{errorText !== undefined && (
				<p role='alert' className='text-sm text-danger'>
					{errorText}
				</p>
			)}
			{errors.position !== undefined && (
				<p role='alert' className='text-sm text-danger'>
					{errors.position}
				</p>
			)}
			{hasPhoto && (
				<TextField
					name='alt'
					maxLength={200}
					value={value.alt}
					onChange={(alt) => onChange({ ...value, alt })}
					isInvalid={errors.alt !== undefined}
				>
					<Label>Описание фото (alt)</Label>
					<Input />
					<Description>Что на фото, для незрячих и поисковиков</Description>
					<FieldError>{errors.alt}</FieldError>
				</TextField>
			)}
		</div>
	)
}
