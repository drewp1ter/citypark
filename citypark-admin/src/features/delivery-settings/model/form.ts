import type { components } from '../../../shared/api/schema'

type Settings = components['schemas']['DeliverySettingsInput']
export type DeliverySettingsForm = Record<keyof Settings, string>
export type DeliverySettingsErrors = Partial<DeliverySettingsForm>
const fields = ['cooking', 'outskirtsLabel', 'outskirtsMin', 'outskirtsMax'] as const
export function toForm(value: Settings): DeliverySettingsForm {
	return {
		cooking: value.cooking,
		outskirtsLabel: value.outskirtsLabel,
		outskirtsMin: String(value.outskirtsMin),
		outskirtsMax: String(value.outskirtsMax),
	}
}
export function validateDeliverySettings(form: DeliverySettingsForm): DeliverySettingsErrors {
	const errors: DeliverySettingsErrors = {}
	for (const name of fields) {
		const value = form[name].trim()
		if (!value) {
			errors[name] = 'поле не должно быть пустым'
			continue
		}
		if (name === 'cooking' || name === 'outskirtsLabel') {
			const max = name === 'cooking' ? 100 : 60
			if ([...value].length > max) errors[name] = `значение должно быть не больше ${max}`
		} else if (!/^\d+$/.test(value)) errors[name] = 'Введите целое число от 0 до 100000'
		else if (Number(value) > 100000) errors[name] = 'значение должно быть не больше 100000'
	}
	if (!errors.outskirtsMin && !errors.outskirtsMax && Number(form.outskirtsMax) < Number(form.outskirtsMin))
		errors.outskirtsMax = 'Максимальная цена должна быть не меньше минимальной'
	return errors
}
/** Convert only after validation succeeds. */
export function toRequest(form: DeliverySettingsForm): Settings {
	return {
		cooking: form.cooking.trim(),
		outskirtsLabel: form.outskirtsLabel.trim(),
		outskirtsMin: Number(form.outskirtsMin.trim()),
		outskirtsMax: Number(form.outskirtsMax.trim()),
	}
}
export function mapServerFields(source: Record<string, string> | undefined): DeliverySettingsErrors {
	const errors: DeliverySettingsErrors = {}
	for (const field of fields) if (source?.[field] !== undefined) errors[field] = source[field]
	return errors
}
