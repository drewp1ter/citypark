import type { Metadata, Viewport } from 'next'
import { Onest, Unbounded } from 'next/font/google'
import { DeliveryProvider } from '@/app/delivery'
import { RegisterServiceWorker } from '@/app/pwa'
import { ThemeProvider } from '@/app/theme'
import { getSiteSettings } from '@/shared/api'
import { SiteSettingsProvider } from '@/shared/config'
import '@/app/styles/globals.css'

const unbounded = Unbounded({
	variable: '--font-unbounded',
	weight: ['400', '500', '600'],
	subsets: ['latin', 'cyrillic'],
	display: 'swap',
})

const onest = Onest({
	variable: '--font-onest',
	weight: ['400', '500', '600', '700'],
	subsets: ['latin', 'cyrillic'],
	display: 'swap',
})

export const metadata: Metadata = {
	title: 'Сити Парк — отель, ресторан и доставка в Белореченске',
	description: 'Отель-люкс на 12 номеров, ресторан и летнее кафе, банкеты и доставка еды. Белореченск, ул. Гоголя, 61.',
	applicationName: 'Сити Парк',
	appleWebApp: { title: 'Сити Парк', statusBarStyle: 'default' },
}

// Browsers read the colour outside our CSS, so it is raw hex copied from tokens.css.
export const viewport: Viewport = {
	themeColor: [
		{ media: '(prefers-color-scheme: light)', color: '#f4f4f1' }, // --bg (light)
		{ media: '(prefers-color-scheme: dark)', color: '#0b0b0c' }, // --bg (dark)
	],
}

export default async function RootLayout({ children }: LayoutProps<'/'>) {
	const settings = await getSiteSettings()

	return (
		<html lang='ru' className={`${unbounded.variable} ${onest.variable}`} suppressHydrationWarning>
			<body>
				<RegisterServiceWorker />
				<a
					href='#content'
					className='focus:rounded-pill focus:bg-inverse-bg focus:text-inverse-text sr-only focus:not-sr-only focus:fixed focus:top-3 focus:left-3 focus:z-50 focus:px-5 focus:py-3 focus:font-semibold'
				>
					Перейти к содержимому
				</a>
				<ThemeProvider>
					<SiteSettingsProvider value={settings}>
						<DeliveryProvider>{children}</DeliveryProvider>
					</SiteSettingsProvider>
				</ThemeProvider>
			</body>
		</html>
	)
}
