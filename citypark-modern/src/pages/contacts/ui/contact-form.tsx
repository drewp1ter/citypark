'use client'

import { useActionState, useState } from 'react'
import { Button, RadioPills, TextareaField, TextField, cardClass } from '@/shared/ui'
import { sendMessage } from '../api/send-message'
import { topics, type ContactState } from '../model/schema'

const initialState: ContactState = { status: 'idle' }

/** "Напишите нам": name, phone or e-mail, topic, message. */
export function ContactForm({ className }: { className?: string }) {
	const [state, formAction, pending] = useActionState(sendMessage, initialState)
	const [topic, setTopic] = useState<string>(topics[0].value)

	const errors = state.status === 'error' ? state.errors : {}
	const values = state.status === 'error' ? state.values : undefined

	return (
		<form
			action={formAction}
			noValidate
			className={cardClass(
				'surface',
				`gap-2.5 rounded-lg p-[18px] md:grid md:grid-cols-2 md:gap-3 md:rounded-xl md:p-8 ${className ?? ''}`
			)}
		>
			<h2 className='font-display text-[22px] font-medium md:col-span-2 md:mb-1.5 md:text-[32px] md:tracking-[-0.02em]'>
				Напишите нам
			</h2>

			{state.status === 'success' ? (
				<p role='status' className='text-text-muted md:col-span-2'>
					Спасибо! Сообщение отправлено, мы ответим в ближайшее время.
				</p>
			) : (
				<>
					<TextField label='Имя' name='name' autoComplete='name' defaultValue={values?.name} error={errors.name} />
					<TextField
						label='Телефон или e-mail'
						name='contact'
						autoComplete='email'
						defaultValue={values?.contact}
						error={errors.contact}
					/>

					<RadioPills
						name='topic'
						legend='Тема'
						value={topic}
						onValueChange={setTopic}
						options={topics}
						className='md:col-span-2'
					/>

					<TextareaField
						label='Сообщение'
						name='message'
						rows={4}
						defaultValue={values?.message}
						error={errors.message}
						className='md:col-span-2'
					/>

					{/* Honeypot: invisible to people, bots tend to fill every field. */}
					<div aria-hidden='true' className='absolute -left-[9999px]'>
						<label>
							Сайт
							<input type='text' name='website' tabIndex={-1} autoComplete='off' />
						</label>
					</div>

					<Button
						type='submit'
						variant='inverse'
						size='xl'
						disabled={pending}
						className='md:bg-accent md:text-on-accent md:col-span-2 md:min-h-[54px]'
					>
						{pending ? 'Отправляем…' : 'Отправить'}
					</Button>
				</>
			)}
		</form>
	)
}
