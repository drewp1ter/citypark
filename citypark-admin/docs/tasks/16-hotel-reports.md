# Task 16: Hotel reports

## Goal
«Отчёты» shows the hotel reports from 1C: a dashboard with a revenue-per-day chart and totals for today, yesterday,
7 days and the period, and the table «Сдача номеров» (revenue by organization and room) for a chosen period. Built so
that a new report is one config entry.

## Depends on
03, and citypark-go task 53.

## Read first
`CLAUDE.md`, `docs/tasks/CONVENTIONS.md`, `src/shared/lib/format.ts`, `../citypark-go/docs/API.md` (section
`### Reports`), the Recharts 3 docs for the installed version (`LineChart`/`AreaChart`, `ResponsiveContainer`).

## Background
These replace «Отчёты для руководителя» of the old admin, hotel part only (restaurant reports are out of scope, ADR
0001). The old admin was used on phones, so design for 390 px first. 1C can be slow (up to 30 s) or down: 502
«Сервис отчётов недоступен»; 404 means 1C is not configured on the server.

## Files to create
- `src/features/reports/api.ts`, `src/features/reports/model/reports.ts` (config, + test),
  `src/features/reports/model/periods.ts` (+ test), `src/features/reports/model/tree.ts` (+ test),
  `src/features/reports/ui/reports-home.tsx`, `ui/daily-chart.tsx`, `ui/period-picker.tsx`, `ui/report-page.tsx`,
  `ui/report-tree.tsx`, `src/routes/_app/reports.$reportId.tsx`, `e2e/reports.spec.ts`

## Files to change
- `package.json`: `recharts`.
- `src/routes/_app/reports.tsx`

## Spec

### Config
```ts
export type ReportConfig = {
	id: string                     // URL segment
	title: string
	endpoint: '/reports/hotel/sales'
	groupBy: readonly string[]     // row fields forming the tree levels, outermost first
	columns: readonly { key: string; title: string; kind: 'count' | 'money' }[]
}
export const reports: readonly ReportConfig[] = [
	{ id: 'hotel-sales', title: 'Сдача номеров', endpoint: '/reports/hotel/sales', groupBy: ['organization', 'room'],
	  columns: [{ key: 'count', title: 'Кол-во', kind: 'count' }, { key: 'sum', title: 'Сумма', kind: 'money' }] },
]
```
`tree.ts`: `buildTree(rows, groupBy, sumKeys)` → nested nodes `{ label, totals: Record<key, number>, children }`,
leaves are the last level; totals summed at every level (numbers, formatted only when rendering).

### Periods
`periods.ts`: `presetRange(preset, today)` for `today`, `yesterday`, `week` (last 7 days incl. today), `month`
(current month to today), `prevMonth`; returns `{ from, to }` as `YYYY-MM-DD`. Period in the URL (`from`, `to`), default
`month`.

### Home (`/reports`)
Period picker (presets as chips + custom range). `GET /reports/hotel/daily` for the period: a line/area chart of
`days` (x: `dd.MM`, y: rubles; tooltip with `formatRub`), themed with the HeroUI accent in both themes; tiles «Сегодня»,
«Вчера», «7 дней», «За период» (`total`) computed from `days` (today/yesterday/7 days only when the period includes
them, else «—»). Below: links to the reports in the config.

### Report page (`/reports/$reportId`)
Period picker, then the tree: on < 768 px an accordion (each level shows its label and totals; tapping opens the next
level), on desktop a table with expandable rows (indent per level) and a total row. Columns from the config;
`money` → `formatRub`, `count` → `toLocaleString('ru-RU')`. Unknown `reportId` → «Отчёт не найден».

### States
Loading: skeleton and the text «Запрашиваем данные из 1С…» (it can take long). 502 → «1С не отвечает. Попробуйте позже.»
with «Повторить». 404 → «Отчёты не настроены на сервере». Empty → «Нет данных за период». Do not refetch on window
focus; cache per period for 5 minutes.

## Tests to write
- `tree.test.ts`: two organizations with rooms → totals per level and grand total; empty rows → `[]`.
- `periods.test.ts`: each preset for a fixed `today` (`2026-10-22`): week → `2026-10-16…2026-10-22`; prevMonth →
  `2026-09-01…2026-09-30`; month → `2026-10-01…2026-10-22`.
- `reports.test.ts`: every config `id` is unique and has at least one column.
- `e2e/reports.spec.ts`: with 1C not configured in dev, `/reports` shows «Отчёты не настроены на сервере» (or the chart
  if it is configured); `/reports/hotel-sales` renders the period picker; `/reports/nope` → «Отчёт не найден». Mock the
  two endpoints with `page.route` in a second test to check the chart tiles and the tree totals at 390 and 1440 px.

## Done when
CONVENTIONS "Done means" passes with `npm run e2e -- e2e/reports.spec.ts`.

## Out of scope
Restaurant reports, export to Excel/PDF, printing.
