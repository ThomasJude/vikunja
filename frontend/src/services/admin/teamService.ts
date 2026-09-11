import {AuthenticatedHTTPFactory, apiV2Url} from '@/helpers/fetcher'
import type {IInviteLinkTeam} from '@/modelTypes/IInviteLink'

export default class AdminTeamService {
	private http = AuthenticatedHTTPFactory()

	async search(q = ''): Promise<{items: IInviteLinkTeam[], total: number}> {
		const {data} = await this.http.get(apiV2Url('admin/teams'), {params: {q, per_page: 100}})
		return {items: data.items, total: data.total}
	}
}
