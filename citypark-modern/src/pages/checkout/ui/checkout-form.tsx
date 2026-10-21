"use client";

import { useActionState, useMemo, useState, type ReactNode } from "react";
import { useCartLines, useCartReady } from "@/entities/cart";
import { priceLines, quoteDelivery, type DeliveryMethod, type DeliveryZone, type Dish } from "@/entities/dish";
import { formatPrice, plural, typo } from "@/shared/lib";
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
} from "@/shared/ui";
import { placeOrder } from "../api/place-order";
import type { OrderState } from "../model/schema";

const initialState: OrderState = { status: "idle" };

const dishWord = (count: number) => plural(count, ["блюдо", "блюда", "блюд"]);

type CheckoutFormProps = {
  /** The whole menu: the cart only stores ids, so the prices are looked up here. */
  dishes: readonly Dish[];
  /** Title row of the page (h1 and the steps). */
  intro: ReactNode;
};

export function CheckoutForm({ dishes, intro }: CheckoutFormProps) {
  const [state, formAction, pending] = useActionState(placeOrder, initialState);
  const errors = state.status === "error" ? state.errors : {};
  const values = state.status === "error" ? state.values : undefined;

  const lines = useCartLines();
  const cartReady = useCartReady();
  const priced = useMemo(() => priceLines(lines, dishes), [lines, dishes]);

  const [method, setMethod] = useState<DeliveryMethod>("delivery");
  const [zone, setZone] = useState<DeliveryZone>("city");
  const [when, setWhen] = useState("asap");
  const [payment, setPayment] = useState("card");
  const [consent, setConsent] = useState(false);

  const quote = quoteDelivery({ method, zone, subtotal: priced.subtotal });
  const total = priced.subtotal + (quote.amount ?? 0);
  const deliveryBadge = quote.label === "уточним" ? "доставка уточнится" : `${quote.label} доставка`;
  const canSubmit = cartReady && priced.items.length > 0 && !pending;

  return (
    <div className="mx-auto w-full max-w-narrow-page px-3 pt-2 pb-4 md:px-6 md:pt-10 md:pb-10">
      {intro}

      {cartReady && priced.items.length === 0 ? (
        <div className={cardClass("surface", "items-start gap-3 rounded-lg p-6 md:rounded-xl md:p-10")}>
          <h2 className="heading text-2xl md:text-[32px]">В корзине пока пусто</h2>
          <p className="text-text-muted">Добавьте блюда из меню, чтобы оформить заказ.</p>
          <Button href="/menu" size="xl" className="mt-2">
            Перейти в меню
          </Button>
        </div>
      ) : (
        <>
          {errors.form && (
            <p role="alert" className="mb-2 rounded-sm border border-danger px-4 py-3 text-danger md:mb-3">
              {errors.form}
            </p>
          )}
          <form
            id="checkout"
            action={formAction}
            noValidate
            className="grid gap-2 md:grid-cols-3 md:items-start md:gap-3"
          >
            <input type="hidden" name="lines" value={JSON.stringify(lines)} />

            {/* Получение */}
            <section
              className={cardClass("surface", "gap-3 rounded-lg p-4 md:col-span-2 md:gap-4 md:rounded-xl md:p-7")}
            >
              <h2 className="text-xl font-bold max-md:sr-only">Получение</h2>
              <RadioPills
                name="method"
                legend="Способ получения"
                variant="segmented"
                value={method}
                onValueChange={(value) => setMethod(value as DeliveryMethod)}
                options={[
                  { value: "delivery", label: "Доставка" },
                  { value: "pickup", label: "Самовывоз" },
                ]}
              />
              <div className="grid gap-3 md:grid-cols-2">
                <Select
                  label="Когда"
                  name="when"
                  options={[
                    { value: "asap", label: "В ближайшее время" },
                    { value: "scheduled", label: "Ко времени…" },
                  ]}
                  value={when}
                  onValueChange={setWhen}
                  error={errors.when}
                />
                {when === "scheduled" && (
                  <TextField
                    label="Время"
                    type="time"
                    name="time"
                    min="10:00"
                    max="23:00"
                    defaultValue={values?.time}
                    error={errors.time}
                  />
                )}
                {method === "delivery" && (
                  <>
                    <Select
                      label="Населённый пункт"
                      name="zone"
                      options={[
                        { value: "city", label: "Белореченск" },
                        { value: "outskirts", label: typo("СОТ / ДНТ (+200–350 ₽)"), textValue: "СОТ / ДНТ" },
                      ]}
                      value={zone}
                      onValueChange={(next) => setZone(next as DeliveryZone)}
                      error={errors.zone}
                    />
                    <TextField
                      label="Улица и дом"
                      name="street"
                      autoComplete="street-address"
                      placeholder="Начните вводить адрес"
                      defaultValue={values?.street}
                      error={errors.street}
                      className="md:col-span-2"
                    />
                    <div className="grid grid-cols-2 gap-2 md:col-span-2 md:gap-3">
                      <TextField
                        label={
                          <>
                            <span className="md:hidden">Кв.</span>
                            <span className="hidden md:inline">Квартира / офис</span>
                          </>
                        }
                        name="apartment"
                        defaultValue={values?.apartment}
                        error={errors.apartment}
                      />
                      <TextField
                        label={
                          <>
                            <span className="md:hidden">Подъезд</span>
                            <span className="hidden md:inline">Подъезд, этаж</span>
                          </>
                        }
                        name="entrance"
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
                "surface",
                "gap-3 rounded-lg p-4 md:col-span-2 md:grid md:grid-cols-2 md:rounded-xl md:p-7",
              )}
            >
              <h2 className="text-xl font-bold max-md:sr-only md:col-span-2 md:mb-1">Контакты и оплата</h2>
              <TextField label="Имя" name="name" autoComplete="name" defaultValue={values?.name} error={errors.name} />
              <PhoneField label="Телефон" name="phone" defaultValue={values?.phone} error={errors.phone} />
              <TextareaField
                label="Пожелания"
                name="comment"
                rows={2}
                placeholder="Без лука, позвонить заранее…"
                defaultValue={values?.comment}
                error={errors.comment}
                className="md:col-span-2"
              />
              <RadioPills
                name="payment"
                legend="Оплата"
                value={payment}
                onValueChange={setPayment}
                className="md:col-span-2"
                options={[
                  {
                    value: "card",
                    label: (
                      <>
                        <span className="md:hidden">Картой</span>
                        <span className="hidden md:inline">Картой курьеру</span>
                      </>
                    ),
                  },
                  { value: "cash", label: "Наличными" },
                  {
                    value: "sbp",
                    label: (
                      <>
                        <span className="md:hidden">СБП</span>
                        <span className="hidden md:inline">СБП онлайн</span>
                      </>
                    ),
                  },
                ]}
              />
            </section>

            {/* Заказ: a dark panel on desktop; on mobile a collapsible summary and the consent row */}
            <aside
              aria-label="Ваш заказ"
              className="flex flex-col gap-2 md:col-start-3 md:row-span-2 md:row-start-1 md:mt-6 md:sticky md:top-6 md:gap-3.5 md:rounded-xl md:bg-inverse-bg md:p-7 md:text-inverse-text"
            >
              <details className="rounded-lg bg-inverse-bg px-[18px] py-1 text-inverse-text md:hidden">
                <summary className="flex min-h-[52px] cursor-pointer items-center justify-between gap-2 font-semibold">
                  <span>
                    Заказ · {priced.count} {dishWord(priced.count)}
                  </span>
                  <span className="text-accent">{deliveryBadge}</span>
                </summary>
                <ul className="mb-3 text-[15px]">
                  {priced.items.map((item) => (
                    <li key={`${item.dish.id}:${item.variant.id}`} className="flex justify-between gap-3 py-1.5">
                      <span>
                        {item.dish.name} × {item.quantity}
                      </span>
                      <span>{formatPrice(item.total)}</span>
                    </li>
                  ))}
                </ul>
              </details>

              <h2 className="heading hidden text-2xl md:block">Заказ</h2>
              <ul className="hidden text-[15px] md:block">
                {priced.items.map((item) => (
                  <li
                    key={`${item.dish.id}:${item.variant.id}`}
                    className="flex justify-between gap-3 border-b border-inverse-border py-[9px]"
                  >
                    <span>
                      {item.dish.name} × {item.quantity}
                    </span>
                    <span>{formatPrice(item.total)}</span>
                  </li>
                ))}
                <li className="flex justify-between gap-3 py-[9px]">
                  <span className="text-inverse-muted">Доставка</span>
                  <span className="font-semibold text-accent">{quote.label}</span>
                </li>
              </ul>
              <div className="hidden items-baseline justify-between md:flex">
                <span>К оплате</span>
                <span className="font-display text-[32px] font-medium">{formatPrice(total)}</span>
              </div>
              {quote.amount === null && (
                <p className="hidden text-[13px] text-inverse-muted md:block">
                  Доставка ({quote.label}) не входит в сумму, стоимость подтвердим по телефону.
                </p>
              )}

              <CheckboxField
                name="consent"
                checked={consent}
                onChange={(event) => setConsent(event.target.checked)}
                error={errors.consent}
                className="px-1.5 py-1 md:p-0 md:text-inverse-muted"
              >
                Согласен с{" "}
                <AppLink href="/privacy" className="tap-area underline md:text-inverse-text">
                  политикой конфиденциальности
                </AppLink>
              </CheckboxField>

              <Button
                type="submit"
                disabled={!canSubmit}
                size="xl"
                className="hidden min-h-14 text-[17px] md:inline-flex"
              >
                {pending ? "Отправляем…" : "Подтвердить заказ"}
              </Button>
              <p className="hidden text-center text-[13px] text-inverse-muted md:block">Перезвоним для подтверждения</p>
            </aside>

            {/* Mobile: the submit button stays at the bottom of the screen */}
            <div className="sticky bottom-0 z-10 -mx-3 bg-linear-to-b from-transparent to-bg to-30% px-3 pt-2.5 pb-3 md:hidden">
              <Button
                type="submit"
                disabled={!canSubmit}
                className="min-h-[58px] w-full justify-between px-6 text-[17px]"
              >
                <span>{pending ? "Отправляем…" : "Подтвердить"}</span>
                <span className="font-display font-medium">{formatPrice(total)}</span>
              </Button>
            </div>
          </form>
        </>
      )}
    </div>
  );
}
