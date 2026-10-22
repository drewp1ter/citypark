import type { Dish } from '@/entities/dish'
import { ThemeToggle } from './theme-toggle'
import { site } from '@/shared/config'
import { AppLink, Button } from '@/shared/ui'
import { CartButton } from './cart-button'
import { CartPill } from './cart-pill'
import { MobileMenu } from './mobile-menu'
import { NavLinks } from './nav-links'
import { PhoneButton } from './phone-button'

const pill = 'flex items-center justify-between rounded-pill border border-border bg-surface'
const logo = 'inline-flex min-h-11 items-center font-display font-semibold'

type SiteHeaderProps = {
	/**
	 * Menu page variant: wider, with a "Корзина [total]" pill instead of the phone, cart and book buttons.
	 * Pass the menu's dishes so the pill can add up the cart.
	 */
	cartDishes?: readonly Dish[]
}

/**
 * Sticky pill header. Compact (logo + buttons + menu sheet) below 1280 px, full navigation from 1280 px (the 6 links + 4 buttons need ~1090 px).
 */
export function SiteHeader({ cartDishes }: SiteHeaderProps) {
	const menuPage = cartDishes !== undefined

	return (
		<header className={`sticky top-0 z-10 px-3 pt-2.5 xl:px-6 xl:pt-4 ${menuPage ? 'bg-bg pb-2 md:pb-4 xl:pb-2' : ''}`}>
			{/* Mobile / tablet */}
			<div className={`${pill} py-1.5 pr-1.5 pl-[18px] xl:hidden`}>
				<AppLink href='/' className={`${logo} text-base`}>
					{site.wordmark}
				</AppLink>
				<div className='flex gap-1.5'>
					<ThemeToggle />
					{menuPage ? <PhoneButton /> : <CartButton />}
					<MobileMenu />
				</div>
			</div>

			{/* Desktop */}
			<div
				className={`${pill} mx-auto hidden gap-6 py-2 pr-2 pl-6 xl:flex ${
					menuPage ? 'max-w-[calc(var(--container-wide)+34px)]' : 'max-w-[calc(var(--container)+34px)]'
				}`}
			>
				<AppLink href='/' className={`${logo} text-lg`}>
					{site.wordmark}
				</AppLink>
				<NavLinks />
				<div className='flex gap-2'>
					<ThemeToggle />
					{menuPage ? (
						<CartPill dishes={cartDishes} />
					) : (
						<>
							<PhoneButton />
							<CartButton />
							<Button href='/rooms' variant='inverse'>
								Забронировать
							</Button>
						</>
					)}
				</div>
			</div>
		</header>
	)
}
