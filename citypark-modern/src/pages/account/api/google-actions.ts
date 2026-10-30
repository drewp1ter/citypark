'use server'

import { startGoogleSignIn as signIn, startGoogleLink as link } from './google'

export async function startGoogleSignIn(): Promise<void> { await signIn() }
export async function startGoogleLink(): Promise<void> { await link() }
