import { Button } from '@heroui/react'
import { useEffect, useState } from 'react'

type InstallPrompt = Event & {
	prompt: () => Promise<void>
	userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>
}

export function InstallApp() {
	const [prompt, setPrompt] = useState<InstallPrompt | null>(null)
	const [installed, setInstalled] = useState(false)
	const [instructions, setInstructions] = useState(false)

	useEffect(() => {
		const display = window.matchMedia('(display-mode: standalone)')
		const sync = () =>
			setInstalled(display.matches || Boolean((navigator as Navigator & { standalone?: boolean }).standalone))
		const capture = (event: Event) => {
			event.preventDefault()
			setPrompt(event as InstallPrompt)
		}
		const complete = () => {
			setInstalled(true)
			setPrompt(null)
		}
		sync()
		display.addEventListener('change', sync)
		window.addEventListener('beforeinstallprompt', capture)
		window.addEventListener('appinstalled', complete)
		return () => {
			display.removeEventListener('change', sync)
			window.removeEventListener('beforeinstallprompt', capture)
			window.removeEventListener('appinstalled', complete)
		}
	}, [])

	async function install() {
		if (!prompt) {
			setInstructions((value) => !value)
			return
		}
		setPrompt(null)
		try {
			await prompt.prompt()
			if ((await prompt.userChoice).outcome === 'accepted') setInstalled(true)
		} catch {
			setInstructions(true)
		}
	}

	if (installed) return null
	return (
		<div className='flex flex-col gap-2'>
			<Button variant='outline' onPress={() => void install()}>
				Установить приложение
			</Button>
			{instructions && (
				<p role='status' className='text-sm text-muted'>
					На iPhone и iPad откройте админку в Safari: «Поделиться» → «На экран Домой». На Android и компьютере выберите
					«Установить приложение» в меню браузера, если этот пункт доступен.
				</p>
			)}
		</div>
	)
}
