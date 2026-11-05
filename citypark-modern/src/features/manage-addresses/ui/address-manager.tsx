'use client'

import type { ManagementProps } from '../model/types'
import { AddressBook } from './address-book'

export function AddressManager(props: ManagementProps) {
	const key =
		props.context.kind === 'account'
			? `${props.context.surface}:${props.context.accountId}`
			: `${props.context.surface}:${props.context.kind}`
	return <AddressBook key={key} {...props} />
}
