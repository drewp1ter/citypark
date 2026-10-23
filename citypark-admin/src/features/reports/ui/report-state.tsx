import { Button, Skeleton } from '@heroui/react'
import type { UseQueryResult } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { ApiError } from '../../../shared/api'

/** Shared loading, error and 404 states of the 1C reports; renders `children(data)` once the data is there. */
export function ReportState<T>({
	query,
	isEmpty,
	children,
}: {
	query: UseQueryResult<T, Error>
	isEmpty?: (data: T) => boolean
	children: (data: T) => ReactNode
}) {
	if (query.isPending) {
		return (
			<div className='flex flex-col gap-3' aria-busy='true' aria-label='Загрузка'>
				<p className='text-muted'>Запрашиваем данные из 1С…</p>
				<Skeleton className='h-24 w-full rounded-lg' />
				<Skeleton className='h-48 w-full rounded-lg' />
			</div>
		)
	}
	if (query.isError) {
		const error = query.error
		if (error instanceof ApiError && error.status === 404) {
			return <p className='text-muted'>Отчёты не настроены на сервере</p>
		}
		const unavailable = error instanceof ApiError && error.status === 502
		return (
			<div role='alert' className='flex flex-col items-start gap-3'>
				<p className='text-danger'>{unavailable ? '1С не отвечает. Попробуйте позже.' : error.message}</p>
				<Button variant='secondary' onPress={() => void query.refetch()}>
					Повторить
				</Button>
			</div>
		)
	}
	if (isEmpty?.(query.data) === true) return <p className='text-muted'>Нет данных за период</p>
	return <>{children(query.data)}</>
}
