'use server'

import { submitLead } from '@/shared/api'
import { isValidRuPhone, phoneDigits } from '@/shared/lib'
import { contactFields, contactSchema, topics, type ContactField, type ContactState } from '../model/schema'

export async function sendMessage(_previous: ContactState, formData: FormData): Promise<ContactState> {
	// Honeypot: the field is hidden from people, so only a bot fills it. Pretend it worked and drop the message.
	if (String(formData.get('website') ?? '') !== '') return { status: 'success' }

	const values = Object.fromEntries(contactFields.map((field) => [field, String(formData.get(field) ?? '')])) as Record<
		ContactField,
		string
	>

	const parsed = contactSchema.safeParse(values)
	if (!parsed.success) {
		const errors: Partial<Record<ContactField, string>> = {}
		for (const issue of parsed.error.issues) {
			const field = issue.path[0] as ContactField
			errors[field] ??= issue.message
		}
		return { status: 'error', errors, values }
	}

	const { contact, topic, ...rest } = parsed.data
	await submitLead('contact', {
		...rest,
		topic: topics.find((item) => item.value === topic)?.label ?? topic,
		contact: isValidRuPhone(contact) ? `+${phoneDigits(contact)}` : contact,
	})
	return { status: 'success' }
}
