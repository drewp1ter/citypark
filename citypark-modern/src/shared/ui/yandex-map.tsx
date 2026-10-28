'use client'

import { useTheme } from 'next-themes'
import { useEffect, useRef, useState, type ReactNode } from 'react'
import { loadYmaps3 } from '@/shared/lib'

type YandexMapProps = {
	apiKey: string
	/** [longitude, latitude] */
	center: readonly [number, number]
	zoom: number
	/** aria-label of the region, e.g. "Карта: ул. Гоголя, 61". */
	label: string
	className?: string
	/** Shown until the map has loaded, and kept if it fails to load. */
	children?: ReactNode
}

type SchemeLayer = InstanceType<typeof ymaps3.YMapDefaultSchemeLayer>

const PIN_PATH = '<path d="M20 10c0 6-8 12-8 12s-8-6-8-12a8 8 0 0 1 16 0z"/><circle cx="12" cy="10" r="3"/>'

/** The accent round pin, the same look as the pin of the placeholder. */
function createPin(): HTMLElement {
	const pin = document.createElement('div')
	pin.className = 'bg-accent text-on-accent flex size-11 items-center justify-center rounded-full'
	pin.style.transform = 'translate(-50%, -50%)'
	pin.setAttribute('aria-hidden', 'true')
	pin.innerHTML = `<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">${PIN_PATH}</svg>`
	return pin
}

/**
 * Yandex Map (JS API v3) with one pin. Loads the API when the box comes near the viewport.
 * The wheel does not zoom it, so the page scrolls over the map.
 */
export function YandexMap({ apiKey, center, zoom, label, className, children }: YandexMapProps) {
	const { resolvedTheme } = useTheme()
	const theme: 'dark' | 'light' = resolvedTheme === 'dark' ? 'dark' : 'light'
	const boxRef = useRef<HTMLDivElement>(null)
	const hostRef = useRef<HTMLDivElement>(null)
	const layerRef = useRef<SchemeLayer | null>(null)
	const themeRef = useRef(theme)
	const [ready, setReady] = useState(false)
	const [lon, lat] = center

	useEffect(() => {
		themeRef.current = theme
		layerRef.current?.update({ theme })
	}, [theme])

	useEffect(() => {
		const box = boxRef.current
		const host = hostRef.current
		if (!box || !host) return

		let map: InstanceType<typeof ymaps3.YMap> | undefined
		let cancelled = false

		const observer = new IntersectionObserver(
			([entry]) => {
				if (!entry?.isIntersecting) return
				observer.disconnect()
				loadYmaps3(apiKey)
					.then((api) => {
						if (cancelled) return
						const { YMap, YMapDefaultSchemeLayer, YMapDefaultFeaturesLayer, YMapMarker } = api
						map = new YMap(host, {
							location: { center: [lon, lat], zoom },
							behaviors: ['drag', 'pinchZoom', 'dblClick'],
						})
						const layer = new YMapDefaultSchemeLayer({ theme: themeRef.current })
						layerRef.current = layer
						map.addChild(layer)
						map.addChild(new YMapDefaultFeaturesLayer({}))
						map.addChild(new YMapMarker({ coordinates: [lon, lat] }, createPin()))
						setReady(true)
					})
					.catch((error: unknown) => console.error('Yandex map failed to load', error))
			},
			{ rootMargin: '200px' }
		)
		observer.observe(box)

		return () => {
			cancelled = true
			observer.disconnect()
			layerRef.current = null
			map?.destroy()
			setReady(false)
		}
	}, [apiKey, lon, lat, zoom])

	return (
		<div ref={boxRef} role='region' aria-label={label} className={`overflow-hidden ${className ?? ''}`}>
			{children}
			<div ref={hostRef} className={`absolute inset-0 ${ready ? '' : 'invisible'}`} />
		</div>
	)
}
