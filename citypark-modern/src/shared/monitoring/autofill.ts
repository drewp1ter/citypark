import { captureException } from '@sentry/nextjs'
import { sanitizeAutofill, type AutofillReason, type AutofillDetails } from './autofill-data'
export type { AutofillReason, AutofillDetails } from './autofill-data'

export function captureAddressAutofillFailure(reason: AutofillReason, details: AutofillDetails): void {
	const error = new Error('Address autofill failed')
	error.name = 'AddressAutofillError'
	captureException(error, {
		tags: { autofill_reason: reason },
		contexts: { autofill: sanitizeAutofill(details) },
		fingerprint: ['address-autofill', details.stage, reason],
		level: [
			'permission_denied',
			'empty_results',
			'unsupported_region',
			'no_town',
			'ambiguous_town',
			'partial',
		].includes(reason)
			? 'warning'
			: 'error',
	})
}
