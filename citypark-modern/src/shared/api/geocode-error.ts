export class GeocodeError extends Error {
	constructor(
		public readonly reason:
			'provider_network' | 'provider_timeout' | 'provider_http' | 'provider_json' | 'provider_shape',
		public readonly httpStatus?: number
	) {
		super(reason)
		this.name = 'GeocodeError'
	}
}
