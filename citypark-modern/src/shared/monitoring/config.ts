import type { ErrorEvent, BrowserOptions } from '@sentry/nextjs'

type StreamedSpanJSON = Parameters<NonNullable<BrowserOptions['beforeSendSpan']>>[0]

/** Invalid configuration fails early, instead of silently changing telemetry volume. */
export function sampleRate(value: string | undefined, development: boolean): number {
	if (value === undefined || value === '') return development ? 1 : 0.1
	const rate = Number(value)
	if (!value.trim() || !Number.isFinite(rate) || rate < 0 || rate > 1) {
		throw new Error('Sentry trace sample rate must be a finite number between 0 and 1')
	}
	return rate
}

/** Anchor the origin: a hostname substring would also match an attacker's host. */
export function backendTraceTargets(apiUrl: string | undefined): RegExp[] {
	if (!apiUrl) return []
	const origin = new URL(apiUrl).origin.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
	return [new RegExp(`^${origin}(?:/|$)`)]
}

/** Keep code locations and identifiers, excluding free-form messages and request data. */
export function scrubError(event: ErrorEvent): ErrorEvent {
	return {
		type: event.type,
		event_id: event.event_id,
		// Prevent ingestion from deriving a location from the sender's network address.
		user: { ip_address: '0.0.0.0' },
		timestamp: event.timestamp,
		platform: event.platform,
		level: event.level,
		release: event.release,
		environment: event.environment,
		sdk: event.sdk,
		debug_meta: event.debug_meta,
		message: event.exception?.values?.length ? undefined : 'Unexpected application error',
		exception: event.exception && {
			values: event.exception.values?.map((exception) => ({
				type: exception.type,
				value: 'Unexpected application error',
				mechanism: exception.mechanism && {
					type: exception.mechanism.type,
					handled: exception.mechanism.handled,
				},
				stacktrace: exception.stacktrace && {
					frames: exception.stacktrace.frames?.map((frame) => ({
						filename: frame.filename?.split(/[?#]/)[0],
						abs_path: frame.abs_path?.split(/[?#]/)[0],
						function: frame.function,
						module: frame.module,
						lineno: frame.lineno,
						colno: frame.colno,
						in_app: frame.in_app,
					})),
				},
			})),
		},
		contexts: event.contexts?.trace
			? {
					trace: {
						trace_id: event.contexts.trace.trace_id,
						span_id: event.contexts.trace.span_id,
						parent_span_id: event.contexts.trace.parent_span_id,
						op: event.contexts.trace.op,
						status: event.contexts.trace.status,
					},
				}
			: undefined,
	}
}

/** SDK 11 streams spans; transaction-only hooks do not filter these payloads. */
export function scrubSpan(span: StreamedSpanJSON): StreamedSpanJSON {
	const op = typeof span.attributes['sentry.op'] === 'string' ? span.attributes['sentry.op'] : 'operation'
	return {
		trace_id: span.trace_id,
		span_id: span.span_id,
		parent_span_id: span.parent_span_id,
		name: op,
		start_timestamp: span.start_timestamp,
		end_timestamp: span.end_timestamp,
		status: span.status,
		is_segment: span.is_segment,
		attributes: { 'sentry.op': op },
	}
}

export const privacyOptions = {
	dataCollection: {
		userInfo: false,
		cookies: false,
		httpHeaders: false,
		httpBodies: [],
		urlQueryParams: false,
		stackFrameVariables: false,
		databaseQueryData: false,
	},
	beforeSend: scrubError,
	beforeSendSpan: scrubSpan,
	beforeBreadcrumb: () => null,
} satisfies Partial<BrowserOptions>
