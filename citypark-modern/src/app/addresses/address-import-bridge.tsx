import { ImportAddresses } from '@/features/import-addresses'
import { getImportAccountAction } from '@/features/import-addresses/index.server'
import type { AddressSurface } from '@/entities/address'

export async function AddressImportBridge({ surface }: { surface: AddressSurface }) {
	const accountId = await getImportAccountAction(surface)
	return <ImportAddresses surface={surface} initialAccountId={accountId} />
}
