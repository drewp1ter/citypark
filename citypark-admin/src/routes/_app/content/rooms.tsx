import { createFileRoute } from '@tanstack/react-router'
import { RoomsPage } from '../../../features/rooms/ui/rooms-page'

export const Route = createFileRoute('/_app/content/rooms')({
	component: RoomsPage,
})
