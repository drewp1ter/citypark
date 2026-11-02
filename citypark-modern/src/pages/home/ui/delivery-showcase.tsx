'use client'

import { useId, useState, type ReactNode } from 'react'
import type { HomepageMenuGroup } from '@/entities/dish'
import { useMenuRefresh } from '@/shared/lib'
import { AppLink, Chip, Section } from '@/shared/ui'
import { DishTeaser } from './dish-teaser'

export function DeliveryShowcase({
	title,
	groups,
	children,
}: {
	title: string
	groups: HomepageMenuGroup[]
	children: ReactNode
}) {
	const [selectedId, setSelectedId] = useState(groups[0]?.category.id)
	const group = groups.find((item) => item.category.id === selectedId) ?? groups[0]
	const panelId = useId()
	useMenuRefresh()
	if (!group) return null
	return (
		<Section
			title={title}
			action={
				<AppLink href='/menu' className='tap-area text-[15px] font-semibold md:hidden'>
					Меню ↗
				</AppLink>
			}
		>
			<nav
				aria-label='Категории доставки на главной'
				className='no-scrollbar mb-3 flex min-w-0 gap-2 overflow-x-auto pb-1 md:mb-5 md:flex-wrap'
			>
				{groups.map(({ category }) => (
					<Chip
						key={category.id}
						selected={category.id === group.category.id}
						aria-controls={panelId}
						onClick={() => setSelectedId(category.id)}
					>
						{category.name}
					</Chip>
				))}
			</nav>
			<div
				id={panelId}
				aria-label={group.category.name}
				role='region'
				className='flex flex-col gap-2 md:grid md:grid-cols-2 md:gap-3 lg:grid-cols-4'
			>
				{group.dishes.map((dish) => (
					<DishTeaser key={dish.id} dish={dish} />
				))}
				{children}
			</div>
			<p className='sr-only' role='status'>
				{group.category.name}
			</p>
		</Section>
	)
}
