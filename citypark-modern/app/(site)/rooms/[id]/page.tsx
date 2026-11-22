import { generateRoomMetadata, RoomPage } from '@/pages/room'

export async function generateMetadata({ params }: PageProps<'/rooms/[id]'>) {
	return generateRoomMetadata((await params).id)
}

export default async function Page({ params, searchParams }: PageProps<'/rooms/[id]'>) {
	return <RoomPage id={(await params).id} searchParams={await searchParams} />
}
