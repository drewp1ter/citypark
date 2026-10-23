import { AlertDialog, Button, Spinner } from '@heroui/react'
import { toastError, toastSuccess } from '../../../shared/ui/toast'
import { useDeleteNews } from '../api'

export type NewsToDelete = { id: number; title: string }

export function DeleteNewsDialog({
	item,
	onClose,
	onDeleted,
}: {
	item: NewsToDelete | null
	onClose: () => void
	onDeleted?: () => void
}) {
	const remove = useDeleteNews()

	function confirm() {
		if (item === null) return
		remove.mutate(item.id, {
			onSuccess: () => {
				toastSuccess('Новость удалена')
				onClose()
				onDeleted?.()
			},
			onError: toastError,
		})
	}

	return (
		<AlertDialog.Backdrop isOpen={item !== null} onOpenChange={(open) => !open && onClose()}>
			<AlertDialog.Container>
				<AlertDialog.Dialog className='sm:max-w-[420px]'>
					<AlertDialog.Header>
						<AlertDialog.Icon status='danger' />
						<AlertDialog.Heading>Удалить новость?</AlertDialog.Heading>
					</AlertDialog.Header>
					<AlertDialog.Body>
						<p>Удалить новость «{item?.title}»? Она сразу исчезнет с сайта.</p>
					</AlertDialog.Body>
					<AlertDialog.Footer>
						<Button slot='close' variant='tertiary'>
							Отмена
						</Button>
						<Button variant='danger' onPress={confirm} isPending={remove.isPending}>
							{({ isPending }) => (
								<>
									{isPending && <Spinner color='current' size='sm' />}
									Удалить
								</>
							)}
						</Button>
					</AlertDialog.Footer>
				</AlertDialog.Dialog>
			</AlertDialog.Container>
		</AlertDialog.Backdrop>
	)
}
