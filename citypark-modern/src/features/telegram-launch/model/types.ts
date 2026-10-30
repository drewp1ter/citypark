export type TelegramWebApp = {
	initData: string
	ready(): void
	expand?(): void
	viewportHeight?: number
	viewportStableHeight?: number
	safeAreaInset?: { top?: number; right?: number; bottom?: number; left?: number }
	contentSafeAreaInset?: { top?: number; right?: number; bottom?: number; left?: number }
	onEvent?(event: string, callback: () => void): void
	offEvent?(event: string, callback: () => void): void
	BackButton?: { show(): void; hide(): void; onClick(callback: () => void): void; offClick(callback: () => void): void }
}

declare global {
	interface Window {
		Telegram?: { WebApp?: TelegramWebApp }
	}
}
