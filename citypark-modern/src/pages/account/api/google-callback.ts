import { completeOAuthCallback } from './oauth'

export function completeGoogleCallback(request: Request) {
	return completeOAuthCallback('google', request)
}

export function completeYandexCallback(request: Request) {
	return completeOAuthCallback('yandex', request)
}
