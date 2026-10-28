/// <reference types="@yandex/ymaps3-types" />

let loading: Promise<typeof ymaps3> | undefined

/** Adds the JS API v3 script once and resolves when `ymaps3.ready` resolves. Rejects on script error. */
export function loadYmaps3(apiKey: string): Promise<typeof ymaps3> {
	if (loading) return loading

	loading = new Promise<typeof ymaps3>((resolve, reject) => {
		const script = document.createElement('script')
		script.src = `https://api-maps.yandex.ru/v3/?apikey=${encodeURIComponent(apiKey)}&lang=ru_RU`
		script.async = true
		script.onload = () => ymaps3.ready.then(() => resolve(ymaps3), reject)
		script.onerror = () => {
			script.remove()
			reject(new Error('Failed to load the Yandex Maps script'))
		}
		document.head.append(script)
	})
	// A failed load must not stick: the next mount tries again.
	loading.catch(() => {
		loading = undefined
	})
	return loading
}
