import type {ITeam} from './ITeam'

export interface ITeamRelation {
	id: number
	childTeamId: number
	admin: boolean
	childTeam: ITeam
	created: Date
}
