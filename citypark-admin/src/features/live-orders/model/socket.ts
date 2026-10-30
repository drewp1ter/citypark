import type { components } from '../../../shared/api/schema'

export type LiveOrderEvent = components['schemas']['LiveOrderEvent']
export type LiveStatus = 'connecting' | 'online' | 'offline'

const EVENT_TYPES: readonly string[] = ['hello', 'order.pending', 'order.paid']

/** 'http://h/api/v1' → 'ws://h/api/v1/orders/live'; https becomes wss. */
export function liveUrl(apiUrl: string): string {
	return apiUrl.replace(/^http/, 'ws').replace(/\/+$/, '') + '/orders/live'
}

/** Reconnect delay in ms: 1 s doubling up to 30 s, plus 0–20 % jitter. */
export function nextDelay(attempt: number, random: () => number = Math.random): number {
	const base = Math.min(30000, 1000 * 2 ** attempt)
	return base + base * 0.2 * random()
}

function parseEvent(data: unknown): LiveOrderEvent | null {
	if (typeof data !== 'string') return null
	let value: unknown
	try {
		value = JSON.parse(data)
	} catch {
		return null
	}
	if (typeof value !== 'object' || value === null || !('type' in value)) return null
	if (typeof value.type !== 'string' || !EVENT_TYPES.includes(value.type)) return null
	return value as LiveOrderEvent
}

type Options = {
	url: string
	getToken: () => string | null
	onEvent: (e: LiveOrderEvent) => void
	onStatus: (s: LiveStatus) => void
	onFailedConnect: () => void
}

/** Keeps a WebSocket to the live orders stream open, reconnecting with backoff. Returns stop(). */
export function connectLiveOrders({ url, getToken, onEvent, onStatus, onFailedConnect }: Options): () => void {
	let socket: WebSocket | null = null
	let timer: ReturnType<typeof setTimeout> | undefined
	let attempt = 0
	let stopped = false

	function open() {
		const token = getToken()
		if (token === null) {
			onStatus('offline')
			return
		}
		onStatus('connecting')
		const ws = new WebSocket(url, ['bearer', token])
		socket = ws
		let opened = false
		ws.onopen = () => {
			opened = true
		}
		ws.onmessage = (message: MessageEvent) => {
			const event = parseEvent(message.data)
			if (event === null) return
			if (event.type === 'hello') {
				attempt = 0
				onStatus('online')
			}
			onEvent(event)
		}
		ws.onclose = () => {
			if (stopped || socket !== ws) return
			socket = null
			onStatus('offline')
			if (!opened) onFailedConnect()
			timer = setTimeout(open, nextDelay(attempt++))
		}
	}

	open()

	return () => {
		stopped = true
		clearTimeout(timer)
		const ws = socket
		socket = null
		ws?.close(1000)
	}
}
