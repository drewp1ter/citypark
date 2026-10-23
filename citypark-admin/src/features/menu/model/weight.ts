const NBSP = String.fromCharCode(0xa0)

/** weight is in kg: 0.35 → «350 г», 1.5 → «1,5 кг». Another measureUnit is shown as given; no weight → ''. */
export function formatWeight(weight: number, measureUnit: string): string {
	if (weight <= 0) return ''
	const unit = measureUnit.trim().toLowerCase()
	if (unit !== '' && unit !== 'kg' && unit !== 'кг') return `${weight} ${measureUnit.trim()}`
	if (weight < 1) return `${Math.round(weight * 1000)}${NBSP}г`
	return `${String(Math.round(weight * 100) / 100).replace('.', ',')}${NBSP}кг`
}
