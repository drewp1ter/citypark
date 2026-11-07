const CACHE = 'citypark-admin-offline-v1'
const OFFLINE = '/offline.html'

self.addEventListener('install', (event) => {
	event.waitUntil(caches.open(CACHE).then((cache) => cache.add(new Request(OFFLINE, { cache: 'reload' }))))
})

self.addEventListener('activate', (event) => {
	event.waitUntil(
		caches
			.keys()
			.then((names) =>
				Promise.all(
					names
						.filter((name) => name.startsWith('citypark-admin-offline-') && name !== CACHE)
						.map((name) => caches.delete(name))
				)
			)
			.then(() => self.clients.claim())
	)
})

self.addEventListener('fetch', (event) => {
	if (event.request.mode !== 'navigate' || new URL(event.request.url).origin !== self.location.origin) return
	event.respondWith(fetch(event.request).catch(() => caches.match(OFFLINE)))
})
