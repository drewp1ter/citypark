export type ScanSection = {
	id: string
	title: string
	/** Files in `public/documents`, one per page of the scan. */
	pages: readonly { src: string; width: number; height: number }[]
}

const pages = (name: string, count: number, width: number, height: number) =>
	Array.from({ length: count }, (_, i) => ({ src: `/documents/${name}-${i + 1}.jpg`, width, height }))

export const scanSections: readonly ScanSection[] = [
	{
		id: 'classification',
		title: 'Выписка из единого реестра объектов классификации в сфере туристической индустрии',
		pages: pages('classification', 2, 1240, 1755),
	},
	{
		id: 'privacy-policy',
		title: 'Политика ООО «Олимп» в отношении обработки персональных данных',
		pages: pages('privacy-policy', 8, 1264, 1752),
	},
]
