import * as Sentry from '@sentry/nextjs'
import { privacyOptions, sampleRate } from '@/shared/monitoring'

Sentry.init({
	...privacyOptions,
	dsn: process.env.NEXT_PUBLIC_SENTRY_DSN,
	enabled: Boolean(process.env.NEXT_PUBLIC_SENTRY_DSN),
	environment:
		process.env.NEXT_PUBLIC_SENTRY_ENVIRONMENT ||
		(process.env.NODE_ENV === 'development' ? 'development' : 'production'),
	release: process.env.NEXT_PUBLIC_SENTRY_RELEASE,
	tracesSampleRate: sampleRate(
		process.env.NEXT_PUBLIC_SENTRY_TRACES_SAMPLE_RATE,
		process.env.NODE_ENV === 'development'
	),
	// The browser calls Next.js; only the Next server calls the backend.
	tracePropagationTargets: [/^\/(?!\/)/],
})

export const onRouterTransitionStart = Sentry.captureRouterTransitionStart
