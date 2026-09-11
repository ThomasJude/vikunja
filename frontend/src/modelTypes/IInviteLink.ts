import type {IAbstract} from './IAbstract'
import type {IUser} from './IUser'

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
	createdBy: IUser | null
	created: Date | null
	updated: Date | null
	token?: string
}
