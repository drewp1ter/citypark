import { site } from '@/shared/config'
import { Button } from '@/shared/ui'

/** "Акции — первыми в соцсетях" (desktop only): lime block with the VK and OK buttons. */
export function SocialCta() {
	const [vk, ok] = site.social

	return (
		<section className='max-w-page mx-auto w-full px-6 pt-6 pb-6 max-md:hidden'>
			<div className='bg-accent text-on-accent flex flex-wrap items-center justify-between gap-5 rounded-xl px-10 py-8'>
				<h2 className='heading text-[28px] tracking-[-0.02em]'>Акции — первыми в соцсетях</h2>
				<div className='flex gap-2'>
					<Button href={vk.href} variant='inverse' size='xl' className='min-h-[50px]'>
						{vk.name}
					</Button>
					<Button href={ok.href} variant='surface' size='xl' className='min-h-[50px] border-0'>
						{ok.name}
					</Button>
				</div>
			</div>
		</section>
	)
}
