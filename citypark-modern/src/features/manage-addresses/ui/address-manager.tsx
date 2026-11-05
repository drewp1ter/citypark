'use client'

import type { ReactNode } from 'react'
import type { ManagementProps } from '../model/types'
import { AddressBook } from './address-book'

export function AddressManager(props: ManagementProps & { importSlot?: ReactNode }) {
	const key =
		props.context.kind === 'account'
			? `${props.context.surface}:${props.context.accountId}`
			: `${props.context.surface}:${props.context.kind}`
	return (
		<>
			{props.importSlot}
			<AddressBook key={key} {...props} />
		</>
	)
}
