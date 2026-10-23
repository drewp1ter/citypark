import { toast } from '@heroui/react'
import { ApiError } from '../api'

/** Shows an error as a toast: the backend's message for an ApiError, a generic text otherwise. */
export function toastError(err: unknown): void {
	toast.danger(err instanceof ApiError ? err.message : 'Что-то пошло не так. Попробуйте ещё раз.')
}

export function toastSuccess(text: string): void {
	toast.success(text)
}
