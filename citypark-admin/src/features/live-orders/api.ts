import { queryOptions, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useRef, useState } from 'react'
import { api, getSession, unwrap } from '../../shared/api'
import { isSendingToIiko } from '../../shared/lib/iiko-status'
import { moscowToday } from '../../shared/lib/today'
import { applyEvent, groupOrders, type Board, type LiveOrder } from './model/board'
import { connectLiveOrders, liveUrl, type LiveOrderEvent, type LiveStatus } from './model/socket'

const apiUrl = import.meta.env.VITE_API_URL ?? 'http://localhost:3000/api/v1'

export function todayOrdersQuery(today: string) {
	return queryOptions({
		queryKey: ['live-orders', today],
		queryFn: async () => {
			const page = await unwrap(api.GET('/orders', { params: { query: { from: today, to: today, pageSize: 100 } } }))
			return page.orders
		},
		refetchInterval: ({ state: { data } }) => {
			// The live stream covers payments. Check for approval and iiko updates through the list API.
			if (data?.some((order) => isSendingToIiko(order.iikoStatus))) return 2000
			return data?.some((order) => order.iikoStatus === 'awaiting_approval') ? 10000 : false
		},
	})
}

/**
 * Today's orders, kept fresh by the WebSocket and list refreshes. `onOrderEvent` gets every event that carries an
 * order, after the list is updated.
 */
export function useLiveOrders(onOrderEvent?: (e: LiveOrderEvent) => void): {
	board: Board
	status: LiveStatus
	isLoading: boolean
	error: unknown
	refetch: () => void
} {
	const queryClient = useQueryClient()
	const today = moscowToday()
	const query = useQuery(todayOrdersQuery(today))
	const [status, setStatus] = useState<LiveStatus>('connecting')
	const handler = useRef(onOrderEvent)
	useEffect(() => {
		handler.current = onOrderEvent
	})

	useEffect(() => {
		return connectLiveOrders({
			url: liveUrl(apiUrl),
			getToken: () => getSession()?.token ?? null,
			onStatus: setStatus,
			onFailedConnect: () => void queryClient.invalidateQueries({ queryKey: ['live-orders'] }),
			onEvent: (e) => {
				// No replay: after every hello the list is loaded again.
				if (e.type === 'hello') {
					void queryClient.invalidateQueries({ queryKey: ['live-orders'] })
					return
				}
				const day = moscowToday()
				queryClient.setQueryData<LiveOrder[]>(todayOrdersQuery(day).queryKey, (orders) =>
					orders === undefined ? orders : applyEvent(orders, e, day)
				)
				void queryClient.invalidateQueries({ queryKey: ['orders'] })
				handler.current?.(e)
			},
		})
	}, [queryClient])

	return {
		board: groupOrders(query.data ?? []),
		status,
		isLoading: query.isPending,
		error: query.isLoadingError ? query.error : null,
		refetch: () => void query.refetch(),
	}
}
