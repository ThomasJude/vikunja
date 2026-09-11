import AbstractService from '@/services/abstractService'
import InviteLinkModel from '@/models/inviteLink'
import type {IInviteLink} from '@/modelTypes/IInviteLink'
import {apiV2Url} from '@/helpers/fetcher'

export interface CreateInviteLinkBody {
	name: string
	teamIds: number[]
	maxUses: number | null
	expiresAt: Date | null
	skipEmailConfirm: boolean
}

export default class AdminInviteLinkService extends AbstractService<IInviteLink> {
	modelFactory(data: Partial<IInviteLink>) {
		return new InviteLinkModel(data)
	}

	async list(page = 1) {
		const {data} = await this.http.get(apiV2Url('admin/invite-links'), {params: {page}})
		this.totalPages = data.total_pages
		return data.items.map((link: Partial<IInviteLink>) => this.modelFactory(link)) as IInviteLink[]
	}

	async createLink(body: CreateInviteLinkBody) {
		const {data} = await this.http.post(apiV2Url('admin/invite-links'), body)
		return this.modelFactory(data)
	}

	async deleteLink(id: number) {
		await this.http.delete(apiV2Url(`admin/invite-links/${id}`))
	}
}
