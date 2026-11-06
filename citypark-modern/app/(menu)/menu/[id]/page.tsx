import { DishPage, generateDishMetadata } from '@/pages/dish'

export async function generateMetadata({ params }: PageProps<'/menu/[id]'>) {
	return generateDishMetadata((await params).id)
}

export default async function Page({ params }: PageProps<'/menu/[id]'>) {
	return <DishPage id={(await params).id} />
}
