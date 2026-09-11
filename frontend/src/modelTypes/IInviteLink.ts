import type {IAbstract} from './IAbstract'

export interface IInviteLinkTeam {
	id: number
	name: string
}

export interface IInviteLink extends IAbstract {
	id: number
	name: string
	teams: IInviteLinkTeam[]
	maxUses: number | null
	uses: number
	expiresAt: Date | null
	skipEmailConfirm: boolean
	createdById: number
	created: Date | null
	updated: Date | null
	token?: string
}
