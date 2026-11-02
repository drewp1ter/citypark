import { getDeliveryTowns, matchDeliveryTown, normalizeLocality } from '@/entities/dish'
import { dadataDebug, reverseGeocode } from '@/shared/api'
import { captureAddressAutofillFailure } from '@/shared/monitoring'
import { serverEnv } from '@/shared/config'
import { postAddressGeolocate } from './address-geolocate-handler'

export const POST = async (request: Request) => {
	const response = await postAddressGeolocate(request, {
		key: () => serverEnv().DADATA_API_KEY,
		geocode: reverseGeocode,
		towns: getDeliveryTowns,
		match: matchDeliveryTown,
		normalize: normalizeLocality,
		debug: dadataDebug,
		report: captureAddressAutofillFailure,
	})

	return response
}
