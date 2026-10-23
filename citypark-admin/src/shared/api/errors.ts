export class ApiError extends Error {
	constructor(
		readonly status: number,
		message: string,
		readonly fields: Record<string, string> = {}
	) {
		super(message)
		this.name = 'ApiError'
	}
}

function defaultMessage(status: number): string {
	if (status === 0) return 'Сервер недоступен. Проверьте соединение.'
	if (status === 401) return 'Требуется авторизация'
	if (status === 403) return 'Недостаточно прав'
	if (status === 404) return 'Не найдено'
	if (status >= 500) return 'Ошибка сервера. Попробуйте ещё раз.'
	return `Не удалось выполнить запрос (код ${status})`
}

function stringFields(value: unknown): Record<string, string> {
	if (typeof value !== 'object' || value === null) return {}
	const fields: Record<string, string> = {}
	for (const [key, text] of Object.entries(value)) {
		if (typeof text === 'string') fields[key] = text
	}
	return fields
}

/** Builds an ApiError from a status and the parsed error body ({statusCode, message, errors}); status 0 is a network failure. */
export function toApiError(status: number, body: unknown): ApiError {
	if (typeof body === 'object' && body !== null) {
		const message = 'message' in body ? body.message : undefined
		if (typeof message === 'string' && message !== '') {
			return new ApiError(status, message, 'errors' in body ? stringFields(body.errors) : {})
		}
	}
	return new ApiError(status, defaultMessage(status))
}
