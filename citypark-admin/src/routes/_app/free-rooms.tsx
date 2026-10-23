import { createFileRoute } from '@tanstack/react-router'
import { FreeRoomsPage } from '../../features/free-rooms/ui/free-rooms-page'

export const Route = createFileRoute('/_app/free-rooms')({
	component: FreeRoomsPage,
})
