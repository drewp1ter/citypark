import { Button, Description, FieldError, Form, Input, Label, Modal, Spinner, TextField } from '@heroui/react'
import { useState, type FormEvent } from 'react'
import { ApiError } from '../../../shared/api'
import { toastError, toastSuccess } from '../../../shared/ui/toast'
import { useCreateDeliveryPrice, useUpdateDeliveryPrice } from '../api'
import { mapServerFields, validateDeliveryPrice, type DeliveryPriceForm } from '../model/form'
import type { DeliveryPrice } from './delivery-prices-page'

type Errors = Partial<Record<keyof DeliveryPriceForm, string>>

/** Add (item is undefined) or edit dialog. Mount it only while open: the form state starts from `item` each time. */
export function DeliveryPriceDialog({ item, onClose }: { item?: DeliveryPrice; onClose: () => void }) {
	const create = useCreateDeliveryPrice()
	const update = useUpdateDeliveryPrice()
	const [form, setForm] = useState<DeliveryPriceForm>({
		name: item?.name ?? '',
		value: item?.value ?? '',
		price: item === undefined ? '' : String(item.price),
	})
	const [errors, setErrors] = useState<Errors>({})
	const isPending = create.isPending || update.isPending

	function set(field: keyof DeliveryPriceForm) {
		return (text: string) => setForm((current) => ({ ...current, [field]: text }))
	}

	function onError(error: Error) {
		if (error instanceof ApiError && error.status === 409) {
			setErrors({ name: 'Такой город уже есть' })
			return
		}
		if (error instanceof ApiError && error.status === 400) setErrors(mapServerFields(error.fields))
		toastError(error)
	}

	function submit(event: FormEvent<HTMLFormElement>) {
		event.preventDefault()
		const found = validateDeliveryPrice(form)
		setErrors(found)
		if (Object.keys(found).length > 0) return
		const options = {
			onSuccess: () => {
				toastSuccess('Сохранено')
				onClose()
			},
			onError,
		}
		if (item === undefined) create.mutate(form, options)
		else update.mutate({ id: item.id, form }, options)
	}

	return (
		<Modal.Backdrop isOpen onOpenChange={(open) => !open && onClose()}>
			<Modal.Container>
				<Modal.Dialog className='sm:max-w-[480px]'>
					<Modal.CloseTrigger />
					<Modal.Header>
						<Modal.Heading>{item === undefined ? 'Добавить город' : 'Изменить город'}</Modal.Heading>
					</Modal.Header>
					<Form onSubmit={submit} validationBehavior='aria' className='flex flex-col'>
						<Modal.Body className='flex flex-col gap-4 p-1'>
							<TextField
								autoFocus
								name='name'
								value={form.name}
								onChange={set('name')}
								isInvalid={errors.name !== undefined}
							>
								<Label>Название города</Label>
								<Input />
								<Description>
									Как клиенты пишут город в адресе, например «Белореченск». Цена подставится, только если название
									совпадает
								</Description>
								<FieldError>{errors.name}</FieldError>
							</TextField>
							<TextField name='value' value={form.value} onChange={set('value')} isInvalid={errors.value !== undefined}>
								<Label>Описание</Label>
								<Input />
								<FieldError>{errors.value}</FieldError>
							</TextField>
							<TextField
								name='price'
								inputMode='decimal'
								value={form.price}
								onChange={set('price')}
								isInvalid={errors.price !== undefined}
							>
								<Label>Цена доставки, ₽</Label>
								<Input />
								<FieldError>{errors.price}</FieldError>
							</TextField>
						</Modal.Body>
						<Modal.Footer>
							<Button slot='close' variant='tertiary'>
								Отмена
							</Button>
							<Button type='submit' variant='primary' isPending={isPending}>
								{({ isPending: pending }) => (
									<>
										{pending && <Spinner color='current' size='sm' />}
										Сохранить
									</>
								)}
							</Button>
						</Modal.Footer>
					</Form>
				</Modal.Dialog>
			</Modal.Container>
		</Modal.Backdrop>
	)
}
