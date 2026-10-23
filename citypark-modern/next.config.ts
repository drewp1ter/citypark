import type { NextConfig } from 'next'

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
	allowedDevOrigins: ['gpubox', '172.19.0.2', '192.168.2.15', '192.168.4.15'],
	images: remotePatterns.length > 0 ? { remotePatterns } : undefined,
	turbopack: {
		rules: {
			'*.css': {
				loaders: ['@tailwindcss/turbopack'],
				as: '*.css',
			},
		},
	},
}

export default nextConfig
