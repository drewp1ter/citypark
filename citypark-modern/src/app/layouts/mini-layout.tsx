import { Suspense, type ReactNode } from 'react'
import { AddressImportBridge } from '../addresses/address-import-bridge'
import { TelegramProvider } from '@/features/telegram-launch'
import { getMiniProfile } from '@/features/telegram-launch'

/** Telegram-only frame; private pages independently check the mini session before fetching content. */
export async function MiniLayout({ children }: { children: ReactNode }) {
	const authenticated = Boolean(await getMiniProfile())
	return (
		<TelegramProvider authenticated={authenticated}>
			<Suspense fallback={null}>
				<AddressImportBridge surface='mini' />
			</Suspense>
			{children}
		</TelegramProvider>
	)
}
