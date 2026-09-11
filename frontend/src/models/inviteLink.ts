import AbstractModel from './abstractModel'
import UserModel from './user'
import type {IUser} from '@/modelTypes/IUser'
import type {IInviteLink, IInviteLinkTeam} from '@/modelTypes/IInviteLink'

export default class InviteLinkModel extends AbstractModel<IInviteLink> implements IInviteLink {
	id = 0
	name = ''
	teams: IInviteLinkTeam[] = []
	maxUses: number | null = null
	uses = 0
	expiresAt: Date | null = null
	skipEmailConfirm = false
	createdById = 0
	createdBy: IUser | null = null
	created: Date | null = null
	updated: Date | null = null
	token?: string

	constructor(data: Partial<IInviteLink> = {}) {
		super()
		this.assignData(data)
		this.createdBy = this.createdBy ? new UserModel(this.createdBy) : null
		this.expiresAt = this.expiresAt ? new Date(this.expiresAt) : null
		this.created = this.created ? new Date(this.created) : null
		this.updated = this.updated ? new Date(this.updated) : null
	}
}
