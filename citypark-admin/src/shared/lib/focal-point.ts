const CENTRE = { x: 50, y: 50 }

function toPercent(ratio: number): number {
	const clamped = Math.min(1, Math.max(0, Number.isFinite(ratio) ? ratio : 0.5))
	return Math.round(clamped * 100)
}

/** Builds a CSS `object-position` from click ratios (0..1): clamps them and rounds to whole percents, "37% 60%". */
export function toPosition(xRatio: number, yRatio: number): string {
	return `${toPercent(xRatio)}% ${toPercent(yRatio)}%`
}

/** Reads "x% y%" back into percents. An empty or invalid string gives the centre. */
export function parsePosition(p: string): { x: number; y: number } {
	const match = /^\s*(\d{1,3}(?:\.\d+)?)%\s+(\d{1,3}(?:\.\d+)?)%\s*$/.exec(p)
	if (match === null) return { ...CENTRE }
	const x = Number(match[1])
	const y = Number(match[2])
	if (x > 100 || y > 100) return { ...CENTRE }
	return { x, y }
}
