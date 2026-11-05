import { loadAddressBook } from '@/entities/address/index.server'
import { getDeliveryTowns } from '@/entities/dish'
import { AddressManager } from '@/features/manage-addresses'
import { serverEnv } from '@/shared/config'
import { Button } from '@/shared/ui'
export async function AddressesPage() {
	const [book, towns] = await Promise.all([loadAddressBook('web'), getDeliveryTowns().catch(() => null)])
	return (
		<main className='mx-auto flex w-full max-w-3xl flex-col gap-6 px-4 py-10'>
			<h1 className='heading text-3xl md:text-4xl'>Мои адреса</h1>
			<Button href='/menu' variant='ghost' className='self-start'>
				В меню
			</Button>
			<AddressManager
				context={book.context}
				initialEntries={book.entries}
				loadError={book.error}
				towns={towns ?? []}
				townsError={towns === null}
				suggestionsEnabled={Boolean(serverEnv().DADATA_API_KEY)}
			/>
		</main>
	)
}
