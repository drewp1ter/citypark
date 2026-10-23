import {
	Button,
	DateField,
	DateRangePicker,
	I18nProvider,
	Label,
	ListBox,
	RangeCalendar,
	SearchField,
	Select,
} from '@heroui/react'
import { parseDate } from '@internationalized/date'
import { useEffect, useRef, useState } from 'react'
import { ORDER_STATUSES, PAYMENT_TYPES, isOrderStatus, isPaymentType, paymentLabel, statusLabel } from '../model/labels'
import { moscowToday, quickRanges, rangeFor } from '../model/quick-ranges'
import { hasFilters, type OrderSearch } from '../model/search'

const ALL = 'all'
const DEBOUNCE_MS = 400

type Props = {
	search: OrderSearch
	/** Changes some filters; the page goes back to 1. */
	onChange: (patch: Partial<OrderSearch>) => void
	onReset: () => void
}

function QueryField({ q, onCommit }: { q: string | undefined; onCommit: (q: string | undefined) => void }) {
	const [text, setText] = useState(q ?? '')
	// What this field last put into the URL, to tell its own changes from Back, a link or «Сбросить».
	const committed = useRef(q)

	useEffect(() => {
		if (q !== committed.current) {
			committed.current = q
			setText(q ?? '')
		}
	}, [q])

	function commit(value: string) {
		const next = value.trim() === '' ? undefined : value.trim()
		if (next === committed.current) return
		committed.current = next
		onCommit(next)
	}

	useEffect(() => {
		const timer = setTimeout(() => commit(text), DEBOUNCE_MS)
		return () => clearTimeout(timer)
		// eslint-disable-next-line react-hooks/exhaustive-deps -- commit only reads refs and the latest onCommit
	}, [text])

	return (
		<SearchField
			name='q'
			value={text}
			onChange={setText}
			onSubmit={commit}
			onClear={() => commit('')}
			className='w-full sm:w-72'
		>
			<Label>Номер, телефон или имя</Label>
			<SearchField.Group>
				<SearchField.SearchIcon />
				<SearchField.Input />
				<SearchField.ClearButton />
			</SearchField.Group>
		</SearchField>
	)
}

function FilterSelect({
	label,
	value,
	options,
	onChange,
}: {
	label: string
	value: string | undefined
	options: { id: string; text: string }[]
	onChange: (id: string | undefined) => void
}) {
	return (
		<Select
			className='w-full sm:w-48'
			value={value ?? ALL}
			onChange={(key) => onChange(key === null || key === ALL ? undefined : String(key))}
		>
			<Label>{label}</Label>
			<Select.Trigger>
				<Select.Value />
				<Select.Indicator />
			</Select.Trigger>
			<Select.Popover>
				<ListBox>
					{[{ id: ALL, text: 'Все' }, ...options].map((o) => (
						<ListBox.Item key={o.id} id={o.id} textValue={o.text}>
							{o.text}
							<ListBox.ItemIndicator />
						</ListBox.Item>
					))}
				</ListBox>
			</Select.Popover>
		</Select>
	)
}

export function OrderFilters({ search, onChange, onReset }: Props) {
	const range = search.from !== undefined && search.to !== undefined
	const value = range ? { start: parseDate(search.from!), end: parseDate(search.to!) } : null

	return (
		<div className='mb-4 flex flex-col gap-3'>
			<div className='flex flex-col flex-wrap gap-3 sm:flex-row sm:items-end'>
				<QueryField q={search.q} onCommit={(q) => onChange({ q })} />
				<I18nProvider locale='ru-RU'>
					<DateRangePicker
						className='w-full sm:w-72'
						value={value}
						onChange={(r) => onChange({ from: r?.start.toString(), to: r?.end.toString() })}
					>
						<Label>Период</Label>
						<DateField.Group>
							<DateField.InputContainer>
								<DateField.Input slot='start'>{(segment) => <DateField.Segment segment={segment} />}</DateField.Input>
								<DateRangePicker.RangeSeparator />
								<DateField.Input slot='end'>{(segment) => <DateField.Segment segment={segment} />}</DateField.Input>
							</DateField.InputContainer>
							<DateField.Suffix>
								<DateRangePicker.Trigger>
									<DateRangePicker.TriggerIndicator />
								</DateRangePicker.Trigger>
							</DateField.Suffix>
						</DateField.Group>
						<DateRangePicker.Popover>
							<RangeCalendar aria-label='Период заказов'>
								<RangeCalendar.Header>
									<RangeCalendar.Heading />
									<RangeCalendar.NavButton slot='previous' />
									<RangeCalendar.NavButton slot='next' />
								</RangeCalendar.Header>
								<RangeCalendar.Grid>
									<RangeCalendar.GridHeader>
										{(day) => <RangeCalendar.HeaderCell>{day}</RangeCalendar.HeaderCell>}
									</RangeCalendar.GridHeader>
									<RangeCalendar.GridBody>{(date) => <RangeCalendar.Cell date={date} />}</RangeCalendar.GridBody>
								</RangeCalendar.Grid>
							</RangeCalendar>
						</DateRangePicker.Popover>
					</DateRangePicker>
				</I18nProvider>
				<FilterSelect
					label='Статус'
					value={search.status}
					options={ORDER_STATUSES.map((id) => ({ id, text: statusLabel(id) }))}
					onChange={(id) => onChange({ status: isOrderStatus(id) ? id : undefined })}
				/>
				<FilterSelect
					label='Оплата'
					value={search.paymentType}
					options={PAYMENT_TYPES.map((id) => ({ id, text: paymentLabel(id) }))}
					onChange={(id) => onChange({ paymentType: isPaymentType(id) ? id : undefined })}
				/>
			</div>
			<div className='flex flex-wrap items-center gap-2'>
				{Object.values(quickRanges).map((r) => (
					<Button key={r.label} size='sm' variant='secondary' onPress={() => onChange(rangeFor(r, moscowToday()))}>
						{r.label}
					</Button>
				))}
				{hasFilters(search) && (
					<Button size='sm' variant='tertiary' onPress={onReset}>
						Сбросить
					</Button>
				)}
			</div>
		</div>
	)
}
