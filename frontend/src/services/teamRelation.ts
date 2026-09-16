import {apiV2Url, AuthenticatedHTTPFactory} from '@/helpers/fetcher'
import TeamModel from '@/models/team'

import type {ITeam} from '@/modelTypes/ITeam'
import type {ITeamRelation} from '@/modelTypes/ITeamRelation'

interface TeamRelationResponse {
	id: number
	child_team_id?: number
	childTeamId?: number
	admin: boolean
	child_team?: ITeam
	childTeam?: ITeam
	created: string
}

interface PaginatedTeamRelations {
	items: TeamRelationResponse[]
	total: number
}

function mapRelation(relation: TeamRelationResponse): ITeamRelation {
	const childTeam = relation.childTeam ?? relation.child_team

	return {
		id: relation.id,
		childTeamId: relation.childTeamId ?? relation.child_team_id ?? childTeam?.id ?? 0,
		admin: relation.admin,
		childTeam: new TeamModel(childTeam),
		created: new Date(relation.created),
	}
}

export default class TeamRelationService {
	http = AuthenticatedHTTPFactory()
	loading = false

	async getAll(teamId: number): Promise<ITeamRelation[]> {
		this.loading = true

		try {
			const {data} = await this.http.get<PaginatedTeamRelations>(
				apiV2Url(`teams/${teamId}/children`),
				{params: {page: 1, per_page: 1000}},
			)

			return data.items.map(mapRelation)
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

	async toggleAdmin(teamId: number, childTeamId: number): Promise<ITeamRelation> {
		this.loading = true

		try {
			const {data} = await this.http.post<TeamRelationResponse>(
				apiV2Url(`teams/${teamId}/children/${childTeamId}/admin`),
			)

			return mapRelation(data)
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
