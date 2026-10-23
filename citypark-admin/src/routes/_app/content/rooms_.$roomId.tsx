import { createFileRoute } from '@tanstack/react-router'
import { RoomEditPage } from '../../../features/rooms/ui/room-edit-page'

// `rooms_` keeps this page out of the list route's tree: it is not rendered inside the list.
export const Route = createFileRoute('/_app/content/rooms_/$roomId')({
	component: RoomRoute,
})

function RoomRoute() {
	const { roomId } = Route.useParams()
	return <RoomEditPage roomId={roomId} />
}
