import { Pagination } from '@heroui/react'

const NEIGHBOURS = 2

export function PagePagination({
	page,
	pageCount,
	onChange,
}: {
	page: number
	pageCount: number
	onChange: (page: number) => void
}) {
	if (pageCount <= 1) return null
	const first = Math.max(1, Math.min(page - NEIGHBOURS, pageCount - NEIGHBOURS * 2))
	const last = Math.min(pageCount, first + NEIGHBOURS * 2)
	const pages = Array.from({ length: last - first + 1 }, (_, i) => first + i)

	return (
		<Pagination aria-label='Страницы'>
			<Pagination.Content>
				<Pagination.Item>
					<Pagination.Previous isDisabled={page <= 1} onPress={() => onChange(page - 1)}>
						<Pagination.PreviousIcon />
						<span>Назад</span>
					</Pagination.Previous>
				</Pagination.Item>
				{pages.map((p) => (
					<Pagination.Item key={p}>
						<Pagination.Link isActive={p === page} onPress={() => onChange(p)} aria-label={`Страница ${p}`}>
							{p}
						</Pagination.Link>
					</Pagination.Item>
				))}
				<Pagination.Item>
					<Pagination.Next isDisabled={page >= pageCount} onPress={() => onChange(page + 1)}>
						<span>Вперёд</span>
						<Pagination.NextIcon />
					</Pagination.Next>
				</Pagination.Item>
			</Pagination.Content>
		</Pagination>
	)
}
