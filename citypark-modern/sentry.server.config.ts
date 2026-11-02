import * as Sentry from '@sentry/nextjs'
import { backendTraceTargets, privacyOptions, sampleRate } from '@/shared/monitoring'

Sentry.init({
	...privacyOptions,
	dsn: process.env.SENTRY_DSN || process.env.NEXT_PUBLIC_SENTRY_DSN,
	enabled: Boolean(process.env.SENTRY_DSN || process.env.NEXT_PUBLIC_SENTRY_DSN),
	environment:
		process.env.NEXT_PUBLIC_SENTRY_ENVIRONMENT ||
		(process.env.NODE_ENV === 'development' ? 'development' : 'production'),
	release: process.env.SENTRY_RELEASE || process.env.NEXT_PUBLIC_SENTRY_RELEASE,
	tracesSampleRate: sampleRate(
		process.env.NEXT_PUBLIC_SENTRY_TRACES_SAMPLE_RATE,
		process.env.NODE_ENV === 'development'
	),
	tracePropagationTargets: backendTraceTargets(process.env.MENU_API_URL),
})
