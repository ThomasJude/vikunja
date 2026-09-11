import type {VikunjaErrorModel} from '@/client/generated/types.gen'
import type {ValidationError} from '@/helpers/parseValidationErrors'
import {HTTPFactory, apiV2Url} from '@/helpers/fetcher'
import InviteLinkModel from '@/models/inviteLink'

export interface InviteRegistrationCredentials {
	username: string
	email: string
	password: string
	language?: string | null
}

export default class InviteRegistrationService {
	private http = HTTPFactory()

	async get(token: string) {
		const {data} = await this.http.get(apiV2Url(`invite-links/${encodeURIComponent(token)}`))
		return new InviteLinkModel(data)
	}

	async register(token: string, credentials: InviteRegistrationCredentials) {
		try {
			await this.http.post(apiV2Url(`invite-links/${encodeURIComponent(token)}/register`), credentials)
		} catch (error) {
			const problem = (error as {response?: {data?: VikunjaErrorModel & ValidationError}})?.response?.data
			if (problem) {
				problem.message = problem.detail ?? problem.message
				problem.invalid_fields = (problem.errors ?? []).flatMap(field =>
					field.location?.startsWith('body.') ? [`${field.location.slice(5)}: ${field.message}`] : [],
				)
			}
			throw error
		}
	}
}
