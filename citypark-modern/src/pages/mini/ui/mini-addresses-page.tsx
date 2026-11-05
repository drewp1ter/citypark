import { ImportAddresses } from '@/features/import-addresses'
import { redirect } from 'next/navigation'
import { loadAddressBook } from '@/entities/address/index.server'
import { getDeliveryTowns } from '@/entities/dish'
import { AddressManager } from '@/features/manage-addresses'
import { serverEnv } from '@/shared/config'
import { Button } from '@/shared/ui'
export async function MiniAddressesPage() {
	const [book, towns] = await Promise.all([loadAddressBook('mini'), getDeliveryTowns().catch(() => null)])
	if (book.context.kind === 'auth-required' && !book.error) redirect('/mini')
	return (
		<main className='mx-auto flex w-full max-w-3xl flex-col gap-6 px-4 py-6 pb-[var(--mini-bottom,env(safe-area-inset-bottom))]'>
			<h1 className='heading text-3xl'>Мои адреса</h1>
			<Button href='/mini/account' variant='ghost' className='self-start'>
				Аккаунт
			</Button>
			<AddressManager
				importSlot={
					<ImportAddresses
						manual
						surface='mini'
						initialAccountId={book.context.kind === 'account' ? book.context.accountId : null}
					/>
				}
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
