// Service worker: shows /offline.html when a page cannot be loaded. Caches nothing else on purpose
// (menu and prices must always be fresh). Bump VERSION when offline.html changes.
const VERSION = 'v1'
const CACHE = `citypark-offline-${VERSION}`
const OFFLINE_URL = '/offline.html'

self.addEventListener('install', (event) => {
	event.waitUntil(
		caches
			.open(CACHE)
			.then((cache) => cache.add(new Request(OFFLINE_URL, { cache: 'reload' })))
			.then(() => self.skipWaiting())
	)
})

self.addEventListener('activate', (event) => {
	event.waitUntil(
		caches
			.keys()
			.then((names) =>
				Promise.all(
					names.filter((name) => name.startsWith('citypark-offline-') && name !== CACHE).map((name) => caches.delete(name))
				)
			)
			.then(() => self.clients.claim())
	)
})

self.addEventListener('fetch', (event) => {
	if (event.request.mode !== 'navigate') return
	if (new URL(event.request.url).pathname === '/account' || new URL(event.request.url).pathname.startsWith('/account/') || new URL(event.request.url).pathname.startsWith('/mini')) return
	event.respondWith(fetch(event.request).catch(() => caches.match(OFFLINE_URL)))
})
