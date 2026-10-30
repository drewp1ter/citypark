import { connection } from 'next/server'
import { getMenu } from '@/entities/dish'
import { getMiniProfile } from '@/features/telegram-launch'
import { getWorkingHours } from '@/shared/api'
import { formatHours } from '@/shared/lib'
import { CartBar, CartPanel } from '@/widgets/cart-summary'
import { MenuCatalog } from '@/widgets/food-ordering'
import { AppLink } from '@/shared/ui'

export async function MiniMenuPage({ categoryId }: { categoryId?: string }) {
	await connection()
	if (!(await getMiniProfile())) return null
	const [menu, hours] = await Promise.all([getMenu(), getWorkingHours()])
	return (
		<div className='flex min-h-[var(--mini-height,100dvh)] flex-col pt-[var(--mini-top,env(safe-area-inset-top))] pr-[var(--mini-right,env(safe-area-inset-right))] pb-[var(--mini-bottom,env(safe-area-inset-bottom))] pl-[var(--mini-left,env(safe-area-inset-left))]'>
			<header className='border-border bg-bg sticky top-0 z-10 border-b px-4 py-3'>
				<p className='font-display text-lg font-semibold'>Сити Парк</p>
				<p className='text-text-muted text-sm'>Доставка · {formatHours(hours)}</p>
				<AppLink href='/mini/account' className='inline-flex min-h-11 items-center underline'>Аккаунт и заказы</AppLink>
			</header>
			<main id='content' className='mx-auto w-full max-w-5xl flex-1 px-3 py-4 md:px-6'>
				<h1 className='heading text-3xl'>Меню</h1>
				<MenuCatalog
					categories={menu.categories}
					dishes={menu.dishes}
					categoryId={categoryId}
					basePath='/mini'
					aside={<CartPanel dishes={menu.dishes} checkoutHref='/mini/checkout' />}
				/>
			</main>
			<CartBar dishes={menu.dishes} checkoutHref='/mini/checkout' mini />
		</div>
	)
}
