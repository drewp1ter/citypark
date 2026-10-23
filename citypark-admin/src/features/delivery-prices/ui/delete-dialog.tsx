import { AlertDialog, Button, Spinner } from '@heroui/react'
import { toastError, toastSuccess } from '../../../shared/ui/toast'
import { useDeleteDeliveryPrice } from '../api'
import type { DeliveryPrice } from './delivery-prices-page'

export function DeleteDialog({ item, onClose }: { item: DeliveryPrice | null; onClose: () => void }) {
	const remove = useDeleteDeliveryPrice()

	function confirm() {
		if (item === null) return
		remove.mutate(item.id, {
			onSuccess: () => {
				toastSuccess('Удалено')
				onClose()
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
						<AlertDialog.Heading>Удалить город?</AlertDialog.Heading>
					</AlertDialog.Header>
					<AlertDialog.Body>
						<p>Удалить «{item?.name}»? Заказы с этим городом станут без цены доставки.</p>
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
