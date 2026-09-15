import {apiV2Url, AuthenticatedHTTPFactory} from '@/helpers/fetcher'
import TeamModel from '@/models/team'

import type {ITeam} from '@/modelTypes/ITeam'

interface PaginatedTeams {
	items: ITeam[]
	total: number
}

export default class TeamRelationService {
	http = AuthenticatedHTTPFactory()
	loading = false

	async getAll(teamId: number): Promise<ITeam[]> {
		this.loading = true

		try {
			const {data} = await this.http.get<PaginatedTeams>(
				apiV2Url(`teams/${teamId}/children`),
				{params: {page: 1, per_page: 1000}},
			)

			return data.items.map(team => new TeamModel(team))
		} finally {
			this.loading = false
		}
	}

	async create(teamId: number, childTeamId: number): Promise<void> {
		this.loading = true

		try {
			await this.http.post(
				apiV2Url(`teams/${teamId}/children`),
				{child_team_id: childTeamId},
			)
		} finally {
			this.loading = false
		}
	}

	async delete(teamId: number, childTeamId: number): Promise<void> {
		this.loading = true

		try {
			await this.http.delete(
				apiV2Url(`teams/${teamId}/children/${childTeamId}`),
			)
		} finally {
			this.loading = false
		}
	}
}
