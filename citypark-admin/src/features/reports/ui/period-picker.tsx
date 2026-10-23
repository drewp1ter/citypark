import { Button, DateField, DateRangePicker, I18nProvider, Label, RangeCalendar } from '@heroui/react'
import { parseDate } from '@internationalized/date'
import { moscowToday } from '../../../shared/lib/today'
import { activePreset, presetRange, PRESETS, type Period } from '../model/periods'

type Props = { period: Period; onChange: (period: Period) => void }

export function PeriodPicker({ period, onChange }: Props) {
	const today = moscowToday()
	const active = activePreset(period, today)
	return (
		<div className='mb-4 flex flex-col gap-3'>
			<div className='flex flex-wrap items-center gap-2' role='group' aria-label='Быстрый выбор периода'>
				{PRESETS.map((p) => (
					<Button
						key={p.id}
						size='sm'
						variant={active === p.id ? 'primary' : 'secondary'}
						aria-pressed={active === p.id}
						onPress={() => onChange(presetRange(p.id, today))}
					>
						{p.label}
					</Button>
				))}
			</div>
			<I18nProvider locale='ru-RU'>
				<DateRangePicker
					className='w-full sm:w-72'
					value={{ start: parseDate(period.from), end: parseDate(period.to) }}
					onChange={(r) => {
						if (r !== null) onChange({ from: r.start.toString(), to: r.end.toString() })
					}}
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
						<RangeCalendar aria-label='Период отчёта'>
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
		</div>
	)
}
