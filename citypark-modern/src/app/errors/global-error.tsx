'use client'

import { useEffect } from 'react'
import { captureException } from '@/shared/monitoring'
import '@/app/styles/globals.css'

/** This fallback also works when the root layout and its providers failed. */
export function GlobalError({ error, retry }: { error: Error & { digest?: string }; retry: () => void }) {
	useEffect(() => {
		captureException(error)
	}, [error])

	return (
		<html lang='ru'>
			<body>
				<main className='bg-bg text-text min-h-screen px-6 py-16'>
					<h1 className='text-2xl font-bold'>Не удалось загрузить страницу</h1>
					<p className='text-text-muted mt-4'>Попробуйте обновить страницу.</p>
					<button
						type='button'
						onClick={retry}
						className='bg-accent text-on-accent focus-visible:outline-accent mt-6 min-h-11 rounded-lg px-5 font-semibold focus-visible:outline-2 focus-visible:outline-offset-2'
					>
						Обновить
					</button>
				</main>
			</body>
		</html>
	)
}
