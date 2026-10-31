import { Button } from '@heroui/react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate, useSearch } from '@tanstack/react-router'
import { PageHeader } from '../../../shared/ui/page-header'
import { campaignsQuery, campaignQuery } from '../api'
import { toMoscowInput } from '../model/form'
import { PromoCodeForm } from './promo-code-form'

function CampaignEditor({ id, onClose }: { id: number; onClose: () => void }) {
	const query = useQuery(campaignQuery(id))
	if (query.isPending) return <p role='status'>Загрузка промокода…</p>
	if (query.isError)
		return (
			<p role='alert'>
				{query.error.message}{' '}
				<Button variant='secondary' onPress={() => void query.refetch()}>
					Повторить
				</Button>{' '}
				<Button variant='secondary' onPress={onClose}>
					Закрыть
				</Button>
			</p>
		)
	return <PromoCodeForm key={id} campaign={query.data} onClose={onClose} />
}

export function PromoCodesPage() {
	const query = useQuery(campaignsQuery)
	const { edit } = useSearch({ from: '/_app/promo-codes' })
	const navigate = useNavigate({ from: '/promo-codes' })
	const close = () => void navigate({ search: {} })
	return (
		<>
			<PageHeader
				title='Промокоды'
				actions={<Button onPress={() => void navigate({ search: { edit: 'new' } })}>Создать промокод</Button>}
			/>
			{edit === 'new' && <PromoCodeForm key='new' onClose={close} />}
			{typeof edit === 'number' && <CampaignEditor key={edit} id={edit} onClose={close} />}
			{query.isPending && <p role='status'>Загрузка промокодов…</p>}
			{query.isError && (
				<p role='alert'>
					{query.error.message}{' '}
					<Button variant='secondary' onPress={() => void query.refetch()}>
						Повторить
					</Button>
				</p>
			)}
			{query.data?.length === 0 && <p>Промокодов пока нет</p>}
			<ul className='flex flex-col gap-3'>
				{query.data?.map((campaign) => (
					<li
						key={campaign.id}
						className='flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border bg-surface p-4'
					>
						<div className='min-w-0 wrap-anywhere'>
							<h2 className='font-semibold'>{campaign.code}</h2>
							<p>
								{campaign.percentage}% · {campaign.enabled ? 'Активен' : 'Отключён'}
							</p>
							<p className='text-sm text-muted'>
								Москва: {campaign.startsAt ? toMoscowInput(campaign.startsAt).replace('T', ' ') : 'Сразу'} —{' '}
								{campaign.endsAt ? toMoscowInput(campaign.endsAt).replace('T', ' ') : 'Без окончания'}
							</p>
						</div>
						<Button
							variant='secondary'
							aria-label={`Изменить ${campaign.code}`}
							onPress={() => void navigate({ search: { edit: campaign.id } })}
						>
							Изменить
						</Button>
					</li>
				))}
			</ul>
		</>
	)
}
