'use server'

import { startOAuth } from './oauth'

export async function startGoogleSignIn(): Promise<void> { await startOAuth('google', 'login') }
export async function startGoogleLink(): Promise<void> { await startOAuth('google', 'link') }
export async function startYandexSignIn(): Promise<void> { await startOAuth('yandex', 'login') }
export async function startYandexLink(): Promise<void> { await startOAuth('yandex', 'link') }
