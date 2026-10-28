import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { connectLiveOrders, liveUrl, nextDelay, type LiveOrderEvent, type LiveStatus } from './socket'

class FakeSocket {
	static instances: FakeSocket[] = []
	onopen: (() => void) | null = null
	onmessage: ((e: { data: unknown }) => void) | null = null
	onclose: (() => void) | null = null
	closeCode: number | undefined
	constructor(
		readonly url: string,
		readonly protocols: string[]
	) {
		FakeSocket.instances.push(this)
	}
	close(code?: number) {
		this.closeCode = code
		this.onclose?.()
	}
	serverOpen() {
		this.onopen?.()
	}
	serverSend(data: unknown) {
		this.onmessage?.({ data })
	}
	serverClose() {
		this.onclose?.()
	}
}

function setup(token: string | null = 'tok') {
	const events: LiveOrderEvent[] = []
	const statuses: LiveStatus[] = []
	const onFailedConnect = vi.fn()
	const stop = connectLiveOrders({
		url: 'ws://h/api/v1/orders/live',
		getToken: () => token,
		onEvent: (e) => events.push(e),
		onStatus: (s) => statuses.push(s),
		onFailedConnect,
	})
	return { events, statuses, onFailedConnect, stop }
}

function last(): FakeSocket {
	const socket = FakeSocket.instances.at(-1)
	if (socket === undefined) throw new Error('no socket was created')
	return socket
}

beforeEach(() => {
	FakeSocket.instances = []
	vi.useFakeTimers()
	vi.stubGlobal('WebSocket', FakeSocket)
	// No reconnect jitter, so the delays are exact.
	vi.spyOn(Math, 'random').mockReturnValue(0)
})

afterEach(() => {
	vi.useRealTimers()
	vi.unstubAllGlobals()
	vi.restoreAllMocks()
})

describe('liveUrl', () => {
	it('swaps the scheme and appends the path', () => {
		expect(liveUrl('http://h/api/v1')).toBe('ws://h/api/v1/orders/live')
		expect(liveUrl('https://h/api/v1/')).toBe('wss://h/api/v1/orders/live')
	})
})

describe('nextDelay', () => {
	it('doubles from one second up to thirty, with up to 20 % jitter', () => {
		expect(nextDelay(0, () => 0)).toBe(1000)
		expect(nextDelay(3, () => 0)).toBe(8000)
		expect(nextDelay(10, () => 0)).toBe(30000)
		expect(nextDelay(0, () => 1)).toBe(1200)
	})
})

describe('connectLiveOrders', () => {
	it('sends the token as a subprotocol and goes online on hello', () => {
		const { statuses, events } = setup()
		expect(last().protocols).toEqual(['bearer', 'tok'])
		expect(statuses).toEqual(['connecting'])
		last().serverOpen()
		last().serverSend('{"type":"hello"}')
		expect(statuses).toEqual(['connecting', 'online'])
		expect(events).toEqual([{ type: 'hello' }])
	})

	it('does not connect without a token', () => {
		const { statuses } = setup(null)
		expect(FakeSocket.instances).toHaveLength(0)
		expect(statuses).toEqual(['offline'])
	})

	it('reconnects after a close and resets the delay on hello', () => {
		const { statuses, onFailedConnect } = setup()
		last().serverOpen()
		last().serverSend('{"type":"hello"}')
		last().serverClose()
		expect(statuses.at(-1)).toBe('offline')
		expect(onFailedConnect).not.toHaveBeenCalled()
		expect(FakeSocket.instances).toHaveLength(1)
		vi.advanceTimersByTime(1200)
		expect(FakeSocket.instances).toHaveLength(2)
		// A second close without hello doubles the delay.
		last().serverOpen()
		last().serverClose()
		vi.advanceTimersByTime(1200)
		expect(FakeSocket.instances).toHaveLength(2)
		vi.advanceTimersByTime(1000)
		expect(FakeSocket.instances).toHaveLength(3)
		// hello brings it back to the first delay.
		last().serverOpen()
		last().serverSend('{"type":"hello"}')
		last().serverClose()
		vi.advanceTimersByTime(1200)
		expect(FakeSocket.instances).toHaveLength(4)
	})

	it('reports a close before open as a failed connect', () => {
		const { onFailedConnect } = setup()
		last().serverClose()
		expect(onFailedConnect).toHaveBeenCalledTimes(1)
	})

	it('ignores bad JSON and unknown types', () => {
		const { events } = setup()
		last().serverOpen()
		last().serverSend('not json')
		last().serverSend('{"type":"order.exploded"}')
		last().serverSend('[1]')
		last().serverSend('null')
		last().serverSend('{"nothing":1}')
		expect(events).toEqual([])
	})

	it('stop() closes with 1000 and stops reconnecting', () => {
		const { stop, onFailedConnect } = setup()
		const socket = last()
		stop()
		expect(socket.closeCode).toBe(1000)
		vi.advanceTimersByTime(60000)
		expect(FakeSocket.instances).toHaveLength(1)
		expect(onFailedConnect).not.toHaveBeenCalled()
	})
})
