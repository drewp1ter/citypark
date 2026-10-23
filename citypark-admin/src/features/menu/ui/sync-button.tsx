import { Button, Spinner, toast } from '@heroui/react'
import { ApiError } from '../../../shared/api'
import { toastError, toastSuccess } from '../../../shared/ui/toast'
import { useSyncMenu } from '../api'

export function SyncButton() {
	const sync = useSyncMenu()

	function run() {
		sync.mutate(undefined, {
			onSuccess: (result) =>
				toastSuccess(`Меню обновлено: категорий ${result.totalCategories}, блюд ${result.totalProducts}`),
			onError: (err) => {
				if (err instanceof ApiError && err.status === 409) toast.warning('Обновление уже идёт, попробуйте через минуту')
				else toastError(err)
			},
		})
	}

	return (
		<Button variant='primary' isDisabled={sync.isPending} onPress={run}>
			{sync.isPending && <Spinner size='sm' color='current' />}
			Обновить меню из iiko
		</Button>
	)
}
