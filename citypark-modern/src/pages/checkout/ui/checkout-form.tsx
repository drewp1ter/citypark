'use client'

import { useActionState, useMemo, useState, type ReactNode } from 'react'
import { useCartLines, useCartReady } from '@/entities/cart'
import { priceLines, type DeliveryMethod, type DeliveryTown, type Dish } from '@/entities/dish'
import { formatPrice, plural } from '@/shared/lib'
import {
	AppLink,
	Button,
	CheckboxField,
	PhoneField,
	RadioPills,
	Select,
	TextareaField,
	TextField,
	cardClass,
} from '@/shared/ui'
import { placeOrder } from '../api/place-order'
import type { OrderState } from '../model/schema'

const initialState: OrderState = { status: 'idle' }

const dishWord = (count: number) => plural(count, ['блюдо', 'блюда', 'блюд'])

type CheckoutFormProps = {
	/** The whole menu: the cart only stores ids, so the prices are looked up here. */
	dishes: readonly Dish[]
	/** Towns with delivery prices from the backend. */
	towns: readonly DeliveryTown[]
	/** The customer came back from Moneta after a failed payment. */
	paymentFailed: boolean
	/** Title row of the page (h1 and the steps). */
	intro: ReactNode
}

export function CheckoutForm({ dishes, towns, paymentFailed, intro }: CheckoutFormProps) {
	const [state, formAction, pending] = useActionState(placeOrder, initialState)
	const errors = state.status === 'error' ? state.errors : {}
	const values = state.status === 'error' ? state.values : undefined

	const lines = useCartLines()
	const cartReady = useCartReady()
	const priced = useMemo(() => priceLines(lines, dishes), [lines, dishes])

	const [method, setMethod] = useState<DeliveryMethod>('delivery')
	const [city, setCity] = useState(towns[0]?.name ?? '')
	const [when, setWhen] = useState('asap')
	const [consent, setConsent] = useState(false)

	const noDelivery = method === 'delivery' && towns.length === 0
	const deliveryPrice = method === 'delivery' ? (towns.find((town) => town.name === city)?.price ?? 0) : 0
	const total = priced.subtotal + deliveryPrice
	const canSubmit = cartReady && priced.items.length > 0 && !pending && !noDelivery

	return (
		<div className='max-w-narrow-page mx-auto w-full px-3 pt-2 pb-4 md:px-6 md:pt-10 md:pb-10'>
			{intro}

			{cartReady && priced.items.length === 0 ? (
				<div className={cardClass('surface', 'items-start gap-3 rounded-lg p-6 md:rounded-xl md:p-10')}>
					<h2 className='heading text-2xl md:text-[32px]'>В корзине пока пусто</h2>
					<p className='text-text-muted'>Добавьте блюда из меню, чтобы оформить заказ.</p>
					<Button href='/menu' size='xl' className='mt-2'>
						Перейти в меню
					</Button>
				</div>
			) : (
				<>
					{paymentFailed && (
						<p role='alert' className='border-danger text-danger mb-2 rounded-sm border px-4 py-3 md:mb-3'>
							Оплата не прошла. Заказ сохранён в корзине: попробуйте ещё раз или позвоните нам.
						</p>
					)}
					{errors.form && (
						<p
							role='alert'
							className='border-danger text-danger mb-2 rounded-sm border px-4 py-3 whitespace-pre-line md:mb-3'
						>
							{errors.form}
						</p>
					)}
					<form
						id='checkout'
						action={formAction}
						noValidate
						className='grid gap-2 md:grid-cols-3 md:items-start md:gap-3'
					>
						<input type='hidden' name='lines' value={JSON.stringify(lines)} />

						{/* Получение */}
						<section
							className={cardClass('surface', 'gap-3 rounded-lg p-4 md:col-span-2 md:gap-4 md:rounded-xl md:p-7')}
						>
							<h2 className='text-xl font-bold max-md:sr-only'>Получение</h2>
							<RadioPills
								name='method'
								legend='Способ получения'
								variant='segmented'
								value={method}
								onValueChange={(value) => setMethod(value as DeliveryMethod)}
								options={[
									{ value: 'delivery', label: 'Доставка' },
									{ value: 'pickup', label: 'Самовывоз' },
								]}
							/>
							{noDelivery && (
								<p role='alert' className='text-danger'>
									Доставка сейчас недоступна, выберите самовывоз.
								</p>
							)}
							<div className='grid gap-3 md:grid-cols-2'>
								<Select
									label='Когда'
									name='when'
									options={[
										{ value: 'asap', label: 'В ближайшее время' },
										{ value: 'scheduled', label: 'Ко времени…' },
									]}
									value={when}
									onValueChange={setWhen}
									error={errors.when}
								/>
								{when === 'scheduled' && (
									<TextField
										label='Время'
										type='time'
										name='time'
										min='10:00'
										max='23:00'
										defaultValue={values?.time}
										error={errors.time}
									/>
								)}
								{method === 'delivery' && !noDelivery && (
									<>
										<Select
											label='Населённый пункт'
											name='city'
											options={towns.map((town) => ({
												value: town.name,
												label: `${town.name} (+${formatPrice(town.price)})`,
												textValue: town.name,
											}))}
											value={city}
											onValueChange={setCity}
											error={errors.city}
										/>
										<TextField
											label='Улица и дом'
											name='street'
											autoComplete='street-address'
											placeholder='Начните вводить адрес'
											defaultValue={values?.street}
											error={errors.street}
											className='md:col-span-2'
										/>
										<div className='grid grid-cols-2 gap-2 md:col-span-2 md:gap-3'>
											<TextField
												label={
													<>
														<span className='md:hidden'>Кв.</span>
														<span className='hidden md:inline'>Квартира / офис</span>
													</>
												}
												name='apartment'
												defaultValue={values?.apartment}
												error={errors.apartment}
											/>
											<TextField
												label={
													<>
														<span className='md:hidden'>Подъезд</span>
														<span className='hidden md:inline'>Подъезд, этаж</span>
													</>
												}
												name='entrance'
												defaultValue={values?.entrance}
												error={errors.entrance}
											/>
										</div>
									</>
								)}
							</div>
						</section>

						{/* Контакты и оплата */}
						<section
							className={cardClass(
								'surface',
								'gap-3 rounded-lg p-4 md:col-span-2 md:grid md:grid-cols-2 md:rounded-xl md:p-7'
							)}
						>
							<h2 className='text-xl font-bold max-md:sr-only md:col-span-2 md:mb-1'>Контакты и оплата</h2>
							<TextField label='Имя' name='name' autoComplete='name' defaultValue={values?.name} error={errors.name} />
							<PhoneField label='Телефон' name='phone' defaultValue={values?.phone} error={errors.phone} />
							<TextareaField
								label='Пожелания'
								name='comment'
								rows={2}
								placeholder='Без лука, позвонить заранее…'
								defaultValue={values?.comment}
								error={errors.comment}
								className='md:col-span-2'
							/>
						</section>

						{/* Заказ: a dark panel on desktop; on mobile a collapsible summary and the consent row */}
						<aside
							aria-label='Ваш заказ'
							className='md:bg-inverse-bg md:text-inverse-text flex flex-col gap-2 md:sticky md:top-6 md:col-start-3 md:row-span-2 md:row-start-1 md:gap-3.5 md:rounded-xl md:p-7'
						>
							<details className='bg-inverse-bg text-inverse-text rounded-lg px-[18px] py-1 md:hidden'>
								<summary className='flex min-h-[52px] cursor-pointer items-center justify-between gap-2 font-semibold'>
									<span>
										Заказ · {priced.count} {dishWord(priced.count)}
									</span>
									<span className='text-accent'>
										{method === 'delivery' ? `доставка ${formatPrice(deliveryPrice)}` : 'самовывоз'}
									</span>
								</summary>
								<ul className='mb-3 text-[15px]'>
									{priced.items.map((item) => (
										<li key={`${item.dish.id}:${item.variant.id}`} className='flex justify-between gap-3 py-1.5'>
											<span>
												{item.dish.name} × {item.quantity}
											</span>
											<span>{formatPrice(item.total)}</span>
										</li>
									))}
								</ul>
							</details>

							<h2 className='heading hidden text-2xl md:block'>Заказ</h2>
							<ul className='hidden text-[15px] md:block'>
								{priced.items.map((item) => (
									<li
										key={`${item.dish.id}:${item.variant.id}`}
										className='border-inverse-border flex justify-between gap-3 border-b py-[9px]'
									>
										<span>
											{item.dish.name} × {item.quantity}
										</span>
										<span>{formatPrice(item.total)}</span>
									</li>
								))}
								<li className='flex justify-between gap-3 py-[9px]'>
									<span className='text-inverse-muted'>Доставка</span>
									<span className='text-accent font-semibold'>{formatPrice(deliveryPrice)}</span>
								</li>
							</ul>
							<div className='hidden items-baseline justify-between md:flex'>
								<span>К оплате</span>
								<span className='font-display text-[32px] font-medium'>{formatPrice(total)}</span>
							</div>
							<CheckboxField
								name='consent'
								checked={consent}
								onChange={(event) => setConsent(event.target.checked)}
								error={errors.consent}
								className='md:text-inverse-muted px-1.5 py-1 md:p-0'
							>
								Согласен с{' '}
								<AppLink href='/privacy' className='tap-area md:text-inverse-text underline'>
									политикой конфиденциальности
								</AppLink>
							</CheckboxField>

							<Button
								type='submit'
								disabled={!canSubmit}
								size='xl'
								className='hidden min-h-14 text-[17px] md:inline-flex'
							>
								{pending ? 'Переходим к оплате…' : 'Оплатить заказ'}
							</Button>
							<p className='text-inverse-muted hidden text-center text-[13px] md:block'>
								Оплата картой или через СБП на странице MONETA.RU
							</p>
						</aside>

						{/* Mobile: the submit button stays at the bottom of the screen */}
						<div className='to-bg sticky bottom-0 z-10 -mx-3 bg-linear-to-b from-transparent to-30% px-3 pt-2.5 pb-3 md:hidden'>
							<Button
								type='submit'
								disabled={!canSubmit}
								className='min-h-[58px] w-full justify-between px-6 text-[17px]'
							>
								<span>{pending ? 'Переходим к оплате…' : 'Оплатить'}</span>
								<span className='font-display font-medium'>{formatPrice(total)}</span>
							</Button>
						</div>
					</form>
				</>
			)}
		</div>
	)
}
