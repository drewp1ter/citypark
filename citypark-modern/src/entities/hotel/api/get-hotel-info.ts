import { guestDocuments } from '../model/hotel.data'

// Reads go through these functions only; swap the bodies when a CMS appears.

export async function getGuestDocuments() {
	return guestDocuments
}
