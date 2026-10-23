import { describe, expect, it } from 'vitest'
import { toApiError } from './errors'

describe('toApiError', () => {
	it('takes message and fields from a validation body', () => {
		const e = toApiError(400, { statusCode: 400, message: 'Ошибка валидации', errors: { phone: 'неверный', n: 5 } })
		expect(e.status).toBe(400)
		expect(e.message).toBe('Ошибка валидации')
		expect(e.fields).toEqual({ phone: 'неверный' })
	})

	it.each([
		[401, undefined, 'Требуется авторизация'],
		[403, null, 'Недостаточно прав'],
		[404, {}, 'Не найдено'],
		[500, '<html>Bad gateway</html>', 'Ошибка сервера. Попробуйте ещё раз.'],
		[0, undefined, 'Сервер недоступен. Проверьте соединение.'],
		[418, undefined, 'Не удалось выполнить запрос (код 418)'],
	])('status %i without a message', (status, body, message) => {
		const e = toApiError(status, body)
		expect(e.message).toBe(message)
		expect(e.fields).toEqual({})
	})

	it('keeps the message of a 409', () => {
		expect(toApiError(409, { statusCode: 409, message: 'Уже существует' }).message).toBe('Уже существует')
	})
})
