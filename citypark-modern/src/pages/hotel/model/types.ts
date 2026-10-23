/** A titled block of text. `html` is sanitized HTML with placeholders filled; '' when the text has not been written yet. */
export type TitledHtml = { title: string; html: string }

export type GuestInfoCopy = {
	heading: string
	intro: string
	sections: readonly TitledHtml[]
}
