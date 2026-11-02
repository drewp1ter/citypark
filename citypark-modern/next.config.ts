import type { NextConfig } from 'next'
import { withSentryConfig } from '@sentry/nextjs/config'

type RemotePattern = NonNullable<NonNullable<NextConfig['images']>['remotePatterns']>[number]

// Host of the menu photos (from iiko); without it remote photos are not used at all.
const menuImageHost = process.env.MENU_IMAGE_HOST

// Origin of the backend's uploaded photos (its PUBLIC_BASE_URL); without it uploaded room photos are not allowed.
const uploads = process.env.UPLOADS_ORIGIN ? new URL(process.env.UPLOADS_ORIGIN) : undefined

const remotePatterns: RemotePattern[] = [
	...(menuImageHost ? [{ protocol: 'https' as const, hostname: menuImageHost }] : []),
	...(uploads
		? [
				{
					protocol: uploads.protocol.replace(':', '') as 'http' | 'https',
					hostname: uploads.hostname,
					port: uploads.port,
					pathname: '/uploads/**',
				},
			]
		: []),
]

const nextConfig: NextConfig = {
	// Dev only: hosts other than localhost that may open the dev server. Without this, Next 16
	// blocks the HMR connection and the page never hydrates (buttons like ThemeToggle do nothing).
	allowedDevOrigins: ['gpubox', '172.19.0.2', '192.168.2.15', '192.168.4.15', 'new.cityparkvip.ru'],
	images: remotePatterns.length > 0 ? { remotePatterns, dangerouslyAllowLocalIP: true } : undefined,
	async headers() {
		return [
			{
				source: '/account/:path*',
				headers: [
					{ key: 'Cache-Control', value: 'private, no-store' },
					{ key: 'Referrer-Policy', value: 'no-referrer' },
				],
			},
			{
				// The worker file must never be cached, or a fixed worker would not reach visitors.
				source: '/sw.js',
				headers: [
					{ key: 'Content-Type', value: 'application/javascript; charset=utf-8' },
					{ key: 'Cache-Control', value: 'no-cache, no-store, must-revalidate' },
				],
			},
		]
	},
	turbopack: {
		rules: {
			'*.css': {
				loaders: ['@tailwindcss/turbopack'],
				as: '*.css',
			},
		},
	},
}

export default withSentryConfig(nextConfig, {
	org: 'citypark-ym',
	project: 'citypark-modern',
	authToken: process.env.SENTRY_AUTH_TOKEN,
	silent: !process.env.SENTRY_AUTH_TOKEN,
	widenClientFileUpload: true,
	sourcemaps: {
		disable: !process.env.SENTRY_AUTH_TOKEN,
		deleteSourcemapsAfterUpload: true,
	},
	release: { name: process.env.NEXT_PUBLIC_SENTRY_RELEASE, setCommits: false },
})
