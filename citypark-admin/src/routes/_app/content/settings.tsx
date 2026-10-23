import { createFileRoute } from '@tanstack/react-router'
import { SettingsPage } from '../../../features/settings/ui/settings-page'

export const Route = createFileRoute('/_app/content/settings')({
	component: SettingsPage,
})
