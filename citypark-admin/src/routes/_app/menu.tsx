import { createFileRoute } from '@tanstack/react-router'
import { parseMenuSearch } from '../../features/menu/model/search'
import { MenuPage } from '../../features/menu/ui/menu-page'

export const Route = createFileRoute('/_app/menu')({
	validateSearch: parseMenuSearch,
	component: MenuPage,
})
