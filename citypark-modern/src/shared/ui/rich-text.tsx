import { cn } from '@/shared/lib'

const base = [
	'[&_p+p]:mt-3 [&_p+ul]:mt-3 [&_p+ol]:mt-3 [&_ul+p]:mt-3 [&_ol+p]:mt-3',
	'[&_strong]:font-bold [&_em]:italic',
	'[&_a]:underline [&_a]:underline-offset-2',
	'[&_ul]:list-disc [&_ol]:list-decimal [&_ul]:pl-5 [&_ol]:pl-5 [&_li+li]:mt-1',
]

const article = [
	'[&_h2]:font-display [&_h2]:mt-8 [&_h2]:mb-3 [&_h2]:text-2xl [&_h2]:leading-[1.15] [&_h2]:font-medium [&_h2]:tracking-[-0.02em]',
	'[&_h3]:mt-6 [&_h3]:mb-2 [&_h3]:text-xl [&_h3]:font-bold',
	'[&_blockquote]:border-border [&_blockquote]:text-text-muted [&_blockquote]:my-4 [&_blockquote]:border-l-4 [&_blockquote]:pl-4',
]

/** Renders sanitized HTML from a content block or a news body. */
export function RichText({
	html,
	variant,
	className,
}: {
	html: string
	variant: 'short' | 'article'
	className?: string
}) {
	return (
		<div
			className={cn(base.join(' '), variant === 'article' && article.join(' '), className)}
			dangerouslySetInnerHTML={{ __html: html }}
		/>
	)
}
