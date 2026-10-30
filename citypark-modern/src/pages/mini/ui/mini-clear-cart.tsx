'use client'
import { useEffect } from 'react'
import { cartActions } from '@/entities/cart'
export function MiniClearCart() {
 useEffect(() => { cartActions.clear() }, [])
 return null
}
