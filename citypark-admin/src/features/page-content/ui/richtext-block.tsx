import { RichTextEditor } from '../../../shared/ui/rich-text-editor'
import { PRICE_PLACEHOLDERS } from '../model/placeholders'

type Props = { label: string; value: string; maxLength: number; onChange: (v: string) => void; error?: string }

export function RichTextBlock({ label, value, maxLength, onChange, error }: Props) {
	return (
		<RichTextEditor
			label={label}
			profile='short'
			maxLength={maxLength}
			placeholders={PRICE_PLACEHOLDERS}
			value={value}
			onChange={onChange}
			error={error}
		/>
	)
}
