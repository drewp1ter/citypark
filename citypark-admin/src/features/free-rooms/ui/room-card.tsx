import { ProgressBar } from '@heroui/react'
import type { RoomType } from '../model/room-types'

export function RoomCard({ type, free }: { type: RoomType; free: number }) {
	return (
		<li className='rounded-xl border border-border bg-surface p-4'>
			<div className='font-medium'>{type.label}</div>
			<div className='mt-1 text-sm text-muted'>
				свободно {free} из {type.total}
			</div>
			<ProgressBar aria-label={type.label} className='mt-3' maxValue={type.total} value={free}>
				<ProgressBar.Track>
					<ProgressBar.Fill />
				</ProgressBar.Track>
			</ProgressBar>
		</li>
	)
}
