import { getDeliveryTowns, matchDeliveryTown } from '@/entities/dish'
import { reverseGeocode } from '@/shared/api'
import { serverEnv } from '@/shared/config'
import { postAddressGeolocate } from './address-geolocate-handler'

export const POST = (request: Request) =>
	postAddressGeolocate(request, {
		key: () => serverEnv().DADATA_API_KEY,
		geocode: reverseGeocode,
		towns: getDeliveryTowns,
		match: matchDeliveryTown,
	})
