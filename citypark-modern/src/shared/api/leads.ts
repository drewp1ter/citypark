export type LeadKind = 'table-booking' | 'order' | 'contact'

/**
 * The one place where every form sends its data (table booking, order, contact form).
 * There is no backend yet, so it only logs the payload on the server.
 *
 * TODO: send to e-mail / Telegram / CRM here (choose by env variable). Only this function changes.
 */
export async function submitLead(kind: LeadKind, payload: Record<string, unknown>): Promise<void> {
	console.info(`[lead:${kind}]`, JSON.stringify(payload))
}
