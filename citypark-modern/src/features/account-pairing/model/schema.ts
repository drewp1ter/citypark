import { z } from 'zod'
import type { CustomerPairing } from '@/shared/customer'

export const pairingCodeSchema = z.string().trim().regex(/^[A-Za-z0-9_-]{22}$/, 'Введите код из Telegram')
export type PairingState = { status: 'idle' | 'pending' | 'completed' | 'canceled' | 'error'; pairing?: CustomerPairing; message?: string; code?: string }
