import Link from 'next/link'
import type { ComponentPropsWithoutRef } from 'react'

const EXTERNAL = /^(tel:|mailto:|https?:\/\/)/
const WEB = /^https?:\/\//

/** Options of next/link that make no sense on a plain <a> (they are dropped there). */
export type LinkExtras = { scroll?: boolean; replace?: boolean; prefetch?: boolean }

type AppLinkProps = Omit<ComponentPropsWithoutRef<'a'>, 'href'> & LinkExtras & { href: string }

/** next/link for pages, a plain <a> for tel:, mailto: and absolute URLs (web links open in a new tab). */
export function AppLink({ href, scroll, replace, prefetch, ...props }: AppLinkProps) {
	if (EXTERNAL.test(href)) {
		const web = WEB.test(href) ? { target: '_blank', rel: 'noopener noreferrer' } : undefined
		return <a href={href} {...web} {...props} />
	}
	return <Link href={href} scroll={scroll} replace={replace} prefetch={prefetch} {...props} />
}
