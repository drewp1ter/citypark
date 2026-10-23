import { Accordion, Button, Skeleton, type Key } from '@heroui/react'
import { useQuery } from '@tanstack/react-query'
import { useBlocker } from '@tanstack/react-router'
import { useCallback, useMemo, useState } from 'react'
import { siteUrl } from '../../../shared/lib/image-src'
import { PageHeader } from '../../../shared/ui/page-header'
import { pageContentQuery, type ContentPage } from '../api'
import { toSections } from '../model/sections'
import { BlockEditor } from './block-editor'

const LEAVE_MESSAGE = 'Есть несохранённые изменения. Уйти без сохранения?'
const TITLES: Record<ContentPage, string> = { home: 'Главная', hotel: 'Отель' }
const SITE_PATHS: Record<ContentPage, string> = { home: '/', hotel: '/hotel' }

/** All sections open on a desktop, only the first one on a phone. */
function initialExpanded(names: string[]): Set<Key> {
	const desktop = window.matchMedia('(min-width: 768px)').matches
	return new Set(desktop ? names : names.slice(0, 1))
}

export function PageContent({ page }: { page: ContentPage }) {
	const query = useQuery(pageContentQuery(page))
	const [dirtyKeys, setDirtyKeys] = useState<ReadonlySet<string>>(new Set())

	const onDirtyChange = useCallback((key: string, dirty: boolean) => {
		setDirtyKeys((current) => {
			if (current.has(key) === dirty) return current
			const next = new Set(current)
			if (dirty) next.add(key)
			else next.delete(key)
			return next
		})
	}, [])

	const hasDirty = dirtyKeys.size > 0
	useBlocker({
		shouldBlockFn: () => !window.confirm(LEAVE_MESSAGE),
		enableBeforeUnload: () => hasDirty,
		disabled: !hasDirty,
	})

	const sections = useMemo(() => toSections(query.data ?? []), [query.data])

	const header = (
		<PageHeader
			title={TITLES[page]}
			actions={
				<a
					href={siteUrl() + SITE_PATHS[page]}
					target='_blank'
					rel='noreferrer'
					className='text-sm text-accent-soft-foreground hover:underline'
				>
					Открыть на сайте ↗
				</a>
			}
		/>
	)

	if (query.isPending) {
		return (
			<>
				{header}
				<Skeleton className='h-64 w-full max-w-3xl rounded-xl' aria-label='Загрузка' />
			</>
		)
	}
	if (query.isError) {
		return (
			<>
				{header}
				<div role='alert' className='flex flex-col items-start gap-3'>
					<p className='text-danger'>{query.error.message}</p>
					<Button variant='secondary' onPress={() => void query.refetch()}>
						Повторить
					</Button>
				</div>
			</>
		)
	}

	return (
		<>
			{header}
			{sections.length === 0 ? (
				<p className='text-muted'>На этой странице нет блоков</p>
			) : (
				<SectionList sections={sections} page={page} dirtyKeys={dirtyKeys} onDirtyChange={onDirtyChange} />
			)}
		</>
	)
}

type ListProps = {
	sections: ReturnType<typeof toSections>
	page: ContentPage
	dirtyKeys: ReadonlySet<string>
	onDirtyChange: (key: string, dirty: boolean) => void
}

function SectionList({ sections, page, dirtyKeys, onDirtyChange }: ListProps) {
	const [expanded, setExpanded] = useState(() => initialExpanded(sections.map((s) => s.name)))
	return (
		<Accordion
			allowsMultipleExpanded
			expandedKeys={expanded}
			onExpandedChange={setExpanded}
			className='w-full max-w-3xl'
		>
			{sections.map((section) => {
				const unsaved = section.blocks.some((b) => dirtyKeys.has(b.key))
				return (
					<Accordion.Item key={section.name} id={section.name}>
						<Accordion.Heading>
							<Accordion.Trigger>
								<span className='flex items-center gap-2'>
									{section.name}
									{unsaved && (
										<span
											role='img'
											aria-label='Есть несохранённые изменения'
											className='inline-block size-2 rounded-full bg-accent'
										/>
									)}
								</span>
								<Accordion.Indicator />
							</Accordion.Trigger>
						</Accordion.Heading>
						<Accordion.Panel>
							<Accordion.Body className='flex flex-col gap-4'>
								{section.blocks.map((block) => (
									<BlockEditor key={block.key} block={block} page={page} onDirtyChange={onDirtyChange} />
								))}
							</Accordion.Body>
						</Accordion.Panel>
					</Accordion.Item>
				)
			})}
		</Accordion>
	)
}
