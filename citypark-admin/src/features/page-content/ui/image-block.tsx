import { useEffect, useState } from 'react'
import { imageSrc } from '../../../shared/lib/image-src'
import { ImageField, type PhotoValue } from '../../../shared/ui/image-field'

type Props = { label: string; value: PhotoValue; onChange: (v: PhotoValue) => void; errors: Record<string, string> }

const DEFAULT_ASPECT = 4 / 3

/** The preview takes the image's own proportions, so the focal point is picked on the whole picture. */
function useNaturalAspect(url: string): number {
	const [loaded, setLoaded] = useState<{ url: string; aspect: number }>()
	useEffect(() => {
		if (url === '') return
		const img = new Image()
		img.onload = () => {
			if (img.naturalWidth > 0 && img.naturalHeight > 0) {
				setLoaded({ url, aspect: img.naturalWidth / img.naturalHeight })
			}
		}
		img.src = imageSrc(url)
	}, [url])
	return loaded?.url === url ? loaded.aspect : DEFAULT_ASPECT
}

export function ImageBlock({ label, value, onChange, errors }: Props) {
	const aspect = useNaturalAspect(value.url)
	return <ImageField label={label} value={value} onChange={onChange} aspect={aspect} errors={errors} />
}
