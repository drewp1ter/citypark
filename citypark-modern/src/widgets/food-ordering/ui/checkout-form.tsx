'use client'

import { useMenuRefresh } from '@/shared/lib'

import { useActionState, useEffect, useMemo, useState, type ReactNode } from 'react'
import { cartActions, useCartLines, useCartReady } from '@/entities/cart'
import { priceLines, townDeliveryPrice, type DeliveryMethod, type DeliveryTown, type Dish } from '@/entities/dish'
import { AddressField, type AddressBookLoad } from '@/entities/address'
import { formatPrice, plural, type WorkingHours } from '@/shared/lib'
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
import { placeMiniOrder, placeOrder } from '../api/place-order'
import type { OrderState } from '../model/schema'
import { useCheckoutQuote } from '../model/use-checkout-quote'
import { PromoCodeField } from './promo-code-field'
import { formatQuotePrice } from '../model/quote-price'
import { useCheckoutLocation } from '../model/use-checkout-location'

import { useCheckoutAddress } from '../model/use-checkout-address'
import { SavedAddressSelector } from './saved-address-selector'
import { useRouter } from 'next/navigation'

const initialState: OrderState = { status: 'idle' }

const dishWord = (count: number) => plural(count, ['блюдо', 'блюда', 'блюд'])

type CheckoutFormProps = {
	addressBook: AddressBookLoad
	/** The whole menu: the cart only stores ids, so the prices are looked up here. */
	dishes: readonly Dish[]
	/** Towns with delivery prices from the backend. */
	towns: readonly DeliveryTown[]
	/** Hours when orders are accepted: the bounds of the «Ко времени» input. */
	hours: WorkingHours
	promoCodesEnabled: boolean
	/** The customer came back from Moneta after a failed payment. */
	paymentFailed: boolean
	/** DaData is configured: «Улица и дом» offers address suggestions. */
	suggestions: boolean
	locationEnabled: boolean
	/** Title row of the page (h1 and the steps). */
	intro: ReactNode
	mini?: boolean
}

export function CheckoutForm({
	addressBook,
	dishes,
	towns,
	hours,
	promoCodesEnabled,
	paymentFailed,
	suggestions,
	locationEnabled,
	intro,
	mini = false,
}: CheckoutFormProps) {
	useMenuRefresh()
	const address = useCheckoutAddress(addressBook, towns)
	const [state, formAction, pending] = useActionState(async (previous: OrderState, data: FormData) => {
		if (!(await address.beforeSubmit(String(data.get('method'))))) return previous
		return (mini ? placeMiniOrder : placeOrder)(previous, data)
	}, initialState)
	const errors = state.status === 'error' ? state.errors : {}
	const values = state.status === 'error' ? state.values : undefined

	const lines = useCartLines()
	const cartReady = useCartReady()
	const priced = useMemo(() => priceLines(lines, dishes), [lines, dishes])

	const [method, setMethod] = useState<DeliveryMethod>(values?.method === 'pickup' ? 'pickup' : 'delivery')
	const [when, setWhen] = useState(values?.when || 'asap')
	const [consent, setConsent] = useState(values?.consent === 'on')
	const { city, street, apartment, entrance } = address.fields
	const router = useRouter()
	useEffect(() => {
		if (!locationEnabled && method === 'delivery' && address.locationAllowed) queueMicrotask(address.fallback)
	}, [locationEnabled, method, address.locationAllowed, address.fallback])
	const location = useCheckoutLocation({
		surface: mini ? 'mini' : 'web',
		enabled: method === 'delivery' && cartReady && priced.items.length > 0 && address.locationAllowed,
		configured: locationEnabled,
		initialTown: '',
		initialStreet: '',
		onDetected: address.detected,
		onFailure: address.fallback,
	})
	const editAddress = (field: keyof typeof address.fields, value: string) => {
		location.takeOver()
		address.edit(field, value)
	}

	const noDelivery = method === 'delivery' && towns.length === 0
	const selectedTown = towns.find((town) => town.name === city)
	const pricing = useCheckoutQuote(
		{ lines, method, city, menu: dishes.map((dish) => [dish.id, dish.available, dish.orderBlocked, dish.variants]) },
		cartReady && priced.items.length > 0 && !noDelivery && (method === 'pickup' || Boolean(selectedTown)),
		state.status === 'error' ? (state.refreshQuote ?? JSON.stringify(state.values)) : undefined,
		mini ? 'mini' : 'web',
		promoCodesEnabled
	)
	const quote = pricing.quote
	const deliveryPrice = quote?.deliveryPrice
	const total = quote?.payableTotal
	const displayMoney = (amount: number | undefined) => (amount === undefined ? '—' : formatQuotePrice(amount))
	const deliveryBecamePaid =
		!!quote?.promoCode &&
		quote.deliveryPrice > 0 &&
		!!selectedTown &&
		townDeliveryPrice(selectedTown, quote.subtotal) === 0
	const canSubmit = address.ready && cartReady && priced.items.length > 0 && !pending && !noDelivery && pricing.ready

	return (
		<div className='max-w-narrow-page mx-auto w-full px-3 pt-2 pb-4 md:px-6 md:pt-10 md:pb-10'>
			{intro}

			{cartReady && priced.items.length === 0 ? (
				<div className={cardClass('surface', 'items-start gap-3 rounded-lg p-6 md:rounded-xl md:p-10')}>
					<h2 className='heading text-2xl md:text-[32px]'>В корзине пока пусто</h2>
					<p className='text-text-muted'>Добавьте блюда из меню, чтобы оформить заказ.</p>
					<Button href={mini ? '/mini' : '/menu'} size='xl' className='mt-2'>
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
						onSubmit={() => location.takeOver()}
						onResetCapture={(event) => {
							// React Aria listens to the native reset event even on controlled fields.
							// Stop it before those listeners can erase a failed submission's draft.
							event.preventDefault()
							event.stopPropagation()
						}}
						noValidate
						className='grid gap-2 md:grid-cols-3 md:items-start md:gap-3'
					>
						<input type='hidden' name='addressContext' value={addressBook.context.kind} />
						<input
							type='hidden'
							name='expectedAccountId'
							value={addressBook.context.kind === 'account' ? addressBook.context.accountId : ''}
						/>
						<input type='hidden' name='lines' value={JSON.stringify(lines)} />
						<input type='hidden' name='promoCode' value={pricing.code} />
						<input type='hidden' name='expectedPayableTotal' value={quote?.payableTotal ?? ''} />

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
								onValueChange={(value) => {
									setMethod(value as DeliveryMethod)
									if (value === 'pickup') location.pickup()
								}}
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
							{method === 'delivery' && location.status && (
								<p role='status' className='text-text-muted'>
									{location.status}
								</p>
							)}
							{method === 'delivery' && (
								<>
									<SavedAddressSelector
										entries={address.entries}
										selected={address.selected}
										towns={towns}
										mini={mini}
										onSelect={(key) => {
											location.takeOver()
											address.select(key)
										}}
									/>
									{addressBook.error && (
										<p role='status'>
											{addressBook.error}{' '}
											<button type='button' className='tap-area underline' onClick={() => router.refresh()}>
												Повторить
											</button>
										</p>
									)}
									<CheckboxField
										name='saveAddress'
										checked={address.save}
										onChange={(event) => address.setSave(event.target.checked)}
									>
										Сохранить адрес
									</CheckboxField>
									{address.save && (
										<>
											<TextField
												label='Название адреса'
												name='addressName'
												value={address.name}
												onChange={(event) => address.setName(event.target.value)}
												error={errors.addressName}
											/>
											<p className='text-text-muted text-sm'>
												Адрес будет сохранён при оформлении заказа, независимо от результата оплаты.
											</p>
										</>
									)}
									{(address.saveError || errors.addressSave) && (
										<p role='alert' className='text-danger'>
											{address.saveError || errors.addressSave}
										</p>
									)}
								</>
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
										min={hours.start}
										max={hours.end}
										defaultValue={values?.time}
										error={errors.time}
									/>
								)}
								{method === 'delivery' && !noDelivery && (
									<>
										<Select
											label='Населённый пункт'
											name='city'
											options={[
												{ value: '', label: 'Выберите населённый пункт', textValue: 'Выберите населённый пункт' },
												...towns.map((town) => ({
													value: town.name,
													label:
														town.freeFrom > 0
															? `${town.name} (+${formatPrice(town.price)}, бесплатно от ${formatPrice(town.freeFrom)})`
															: `${town.name} (+${formatPrice(town.price)})`,
													textValue: town.name,
												})),
											]}
											value={city}
											onValueChange={(value) => {
												editAddress('city', value)
											}}
											error={errors.city}
										/>
										<AddressField
											label='Улица и дом'
											name='street'
											town={city}
											suggestions={suggestions}
											value={street}
											onValueChange={(value) => {
												if (value !== street) editAddress('street', value)
											}}
											onManualEdit={() => {
												location.takeOver()
												address.takeOver()
											}}
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
												value={apartment}
												onChange={(event) => editAddress('apartment', event.target.value)}
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
												value={entrance}
												onChange={(event) => editAddress('entrance', event.target.value)}
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
										{method === 'delivery' ? `доставка ${displayMoney(deliveryPrice)}` : 'самовывоз'}
									</span>
								</summary>
								<ul className='mb-3 text-[15px]'>
									{priced.items.map((item) => (
										<li key={`${item.dish.id}:${item.variant.id}`} className='flex justify-between gap-3 py-1.5'>
											<span>
												{item.dish.name} × {item.quantity}
											</span>
											<span>{formatPrice(item.total)}</span>
											{item.dish.orderBlocked && (
												<button
													type='button'
													aria-label={`Удалить из корзины: ${item.dish.name}`}
													className='min-h-11 min-w-11'
													onClick={() =>
														cartActions.setQuantity({ dishId: item.dish.id, variantId: item.variant.id }, 0)
													}
												>
													×
												</button>
											)}
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
										{item.dish.orderBlocked && (
											<button
												type='button'
												aria-label={`Удалить из корзины: ${item.dish.name}`}
												className='min-h-11 min-w-11'
												onClick={() => cartActions.setQuantity({ dishId: item.dish.id, variantId: item.variant.id }, 0)}
											>
												×
											</button>
										)}
									</li>
								))}
							</ul>
							{promoCodesEnabled && (
								<PromoCodeField
									text={pricing.text}
									onChange={pricing.setText}
									onRemove={pricing.remove}
									pending={pricing.pending}
									error={pricing.promoError || (!pricing.ready ? errors.promoCode : undefined)}
									disabled={pending}
								/>
							)}
							{(pricing.error ||
								(!pricing.ready && errors.expectedPayableTotal) ||
								(!promoCodesEnabled && errors.promoCode)) && (
								<p role='alert' className='bg-surface text-danger rounded-sm p-3 text-sm'>
									{pricing.error || errors.expectedPayableTotal || errors.promoCode}
								</p>
							)}
							<dl
								aria-label='Расчёт заказа'
								className='bg-inverse-bg text-inverse-text flex flex-col gap-2 rounded-lg p-4 md:p-0'
							>
								<div className='flex justify-between gap-2'>
									<dt>Блюда</dt>
									<dd>{displayMoney(quote?.subtotal)}</dd>
								</div>
								{quote?.promoCode && (
									<div className='flex justify-between gap-2'>
										<dt>Скидка по промокоду</dt>
										<dd>{formatQuotePrice(0 - quote.discount)}</dd>
									</div>
								)}
								<div className='flex justify-between gap-2'>
									<dt>Доставка</dt>
									<dd>{displayMoney(deliveryPrice)}</dd>
								</div>
								<div className='flex items-baseline justify-between gap-2 font-semibold'>
									<dt>Итого</dt>
									<dd className='md:font-display md:text-[32px] md:font-medium'>{displayMoney(total)}</dd>
								</div>
							</dl>
							{deliveryBecamePaid && (
								<p role='status' className='bg-surface text-text rounded-sm p-3 text-sm'>
									После скидки доставка стала платной
								</p>
							)}
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
						<div
							className={`to-bg sticky z-10 -mx-3 bg-linear-to-b from-transparent to-30% px-3 pt-2.5 pb-3 md:hidden ${mini ? 'bottom-[var(--mini-bottom)]' : 'bottom-0'}`}
						>
							<Button
								type='submit'
								disabled={!canSubmit}
								className='min-h-[58px] w-full justify-between px-6 text-[17px]'
							>
								<span>{pending ? 'Переходим к оплате…' : 'Оплатить'}</span>
								<span className='font-display font-medium'>{displayMoney(total)}</span>
							</Button>
						</div>
					</form>
				</>
			)}
		</div>
	)
}
