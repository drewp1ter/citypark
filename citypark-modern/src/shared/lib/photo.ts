/** An image from /public/images with its alt text (copied from the references). */
export type Photo = {
	src: string
	alt: string
	/** CSS object-position when the default centre crop cuts the wrong part ("right center"). */
	position?: string
}

/** Used where the owner has not sent a photo yet. */
export const PLACEHOLDER_PHOTO = '/images/placeholder.svg'
