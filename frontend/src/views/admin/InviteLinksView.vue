<template>
	<Card :title="$t('admin.inviteLinks.title')">
		<XButton
			class="mbe-4"
			@click="openCreate"
		>
			{{ $t('admin.inviteLinks.create') }}
		</XButton>
		<p v-if="loading">
			{{ $t('misc.loading') }}
		</p>
		<p v-else-if="links.length === 0">
			{{ $t('admin.inviteLinks.empty') }}
		</p>
		<div
			v-else
			class="overflow-x-auto"
		>
			<table class="table has-actions is-striped is-hoverable is-fullwidth">
				<thead>
					<tr>
						<th>{{ $t('admin.inviteLinks.name') }}</th>
						<th>{{ $t('admin.inviteLinks.teams') }}</th>
						<th>{{ $t('admin.inviteLinks.uses') }}</th>
						<th>{{ $t('admin.inviteLinks.expiresAt') }}</th>
						<th>{{ $t('admin.inviteLinks.createdBy') }}</th>
						<th>{{ $t('task.attributes.created') }}</th>
						<th />
					</tr>
				</thead>
				<tbody>
					<tr
						v-for="link in links"
						:key="link.id"
					>
						<td>{{ link.name }}</td>
						<td>{{ link.teams.map(team => team.name).join(', ') }}</td>
						<td>{{ link.uses }} / {{ link.maxUses ?? '∞' }}</td>
						<td>
							<TimeDisplay
								v-if="link.expiresAt"
								:date="link.expiresAt"
							/>
							<template v-else>
								{{ $t('admin.inviteLinks.neverExpires') }}
							</template>
						</td>
						<td>#{{ link.createdById }}</td>
						<td>
							<TimeDisplay
								v-if="link.created"
								:date="link.created"
							/>
						</td>
						<td class="actions">
							<XButton
								variant="secondary"
								danger
								@click="pendingDelete = link"
							>
								{{ $t('misc.delete') }}
							</XButton>
						</td>
					</tr>
				</tbody>
			</table>
		</div>
		<PaginationEmit
			v-if="totalPages > 1"
			:total-pages="totalPages"
			:current-page="page"
			@pageChanged="loadLinks"
		/>

		<Modal
			v-if="createOpen"
			variant="hint-modal"
			@close="closeCreate"
		>
			<Card
				class="has-no-shadow"
				:title="$t('admin.inviteLinks.create')"
			>
				<template v-if="createdUrl">
					<Message
						variant="warning"
						class="mbe-4"
					>
						{{ $t('admin.inviteLinks.shownOnce') }}
					</Message>
					<FormField :label="$t('admin.inviteLinks.url')">
						<template #default="{id}">
							<FormInput
								:id="id"
								:model-value="createdUrl"
								readonly
								@focus="selectLink"
							/>
						</template>
					</FormField>
					<XButton @click="copy(createdUrl)">
						{{ $t(copied ? 'admin.inviteLinks.copied' : 'admin.inviteLinks.copy') }}
					</XButton>
				</template>
				<form
					v-else
					id="invite-create-form"
					@submit.prevent="submitCreate"
				>
					<FormField
						id="invite-name"
						v-model="form.name"
						:label="$t('admin.inviteLinks.name')"
						required
						maxlength="250"
					/>
					<FormField :label="$t('admin.inviteLinks.teams')">
						<template #default="{id}">
							<Multiselect
								:id="id"
								v-model="selectedTeams"
								multiple
								show-empty
								label="name"
								:search-results="teamResults"
								:loading="loadingTeams"
								@search="searchTeams"
							/>
						</template>
					</FormField>
					<FormField
						id="invite-max-uses"
						v-model="form.maxUses"
						type="number"
						min="1"
						step="1"
						:label="$t('admin.inviteLinks.maxUses')"
						:placeholder="$t('admin.inviteLinks.unlimited')"
					/>
					<FormField
						id="invite-expiry"
						v-model="form.expiresAt"
						type="datetime-local"
						:min="minimumExpiry"
						:label="$t('admin.inviteLinks.expiresAt')"
					/>
					<FormCheckbox
						v-model="form.skipEmailConfirm"
						:label="$t('admin.inviteLinks.skipEmailConfirm')"
					/>
				</form>
				<template #footer>
					<XButton
						variant="tertiary"
						:disabled="creating"
						@click="closeCreate"
					>
						{{ $t(createdUrl ? 'misc.close' : 'misc.cancel') }}
					</XButton>
					<XButton
						v-if="!createdUrl"
						type="submit"
						form="invite-create-form"
						:loading="creating"
						:disabled="!form.name.trim()"
					>
						{{ $t('admin.inviteLinks.create') }}
					</XButton>
				</template>
			</Card>
		</Modal>
		<Modal
			v-if="pendingDelete"
			:loading="deleting"
			@close="pendingDelete = null"
			@submit="deleteLink"
		>
			<template #header>
				{{ $t('admin.inviteLinks.delete') }}
			</template>
			<template #text>
				{{ $t('admin.inviteLinks.deleteConfirm', {name: pendingDelete.name}) }}
			</template>
		</Modal>
	</Card>
</template>

<script setup lang="ts">
import {onMounted, reactive, ref} from 'vue'
import {useClipboard, useDebounceFn} from '@vueuse/core'
import {formatDate} from '@/helpers/time/formatDate'
import {useI18n} from 'vue-i18n'
import Card from '@/components/misc/Card.vue'
import Modal from '@/components/misc/Modal.vue'
import Message from '@/components/misc/Message.vue'
import FormField from '@/components/input/FormField.vue'
import FormInput from '@/components/input/FormInput.vue'
import FormCheckbox from '@/components/input/FormCheckbox.vue'
import Multiselect from '@/components/input/Multiselect.vue'
import TimeDisplay from '@/components/misc/TimeDisplay.vue'
import PaginationEmit from '@/components/misc/PaginationEmit.vue'
import AdminInviteLinkService from '@/services/admin/inviteLinkService'
import AdminTeamService from '@/services/admin/teamService'
import type {IInviteLink, IInviteLinkTeam} from '@/modelTypes/IInviteLink'
import {useConfigStore} from '@/stores/config'
import {useTitle} from '@/composables/useTitle'
import {error, success} from '@/message'

const {t} = useI18n()
useTitle(() => t('admin.inviteLinks.title'))
const configStore = useConfigStore()
const service = new AdminInviteLinkService()
const teamService = new AdminTeamService()
const {copy, copied} = useClipboard({legacy: true})
const links = ref<IInviteLink[]>([])
const page = ref(1)
const totalPages = ref(0)
const loading = ref(false)
const createOpen = ref(false)
const creating = ref(false)
const createdUrl = ref('')
const pendingDelete = ref<IInviteLink | null>(null)
const deleting = ref(false)
const selectedTeams = ref<IInviteLinkTeam[]>([])
const teamResults = ref<IInviteLinkTeam[]>([])
const loadingTeams = ref(false)
const minimumExpiry = ref('')
let initialTeams: IInviteLinkTeam[] = []
let teamCount = 0
const form = reactive({name: '', maxUses: '' as string | number, expiresAt: '', skipEmailConfirm: true})

async function loadLinks(nextPage = page.value) {
	page.value = nextPage
	loading.value = true
	try {
		links.value = await service.list(nextPage)
		totalPages.value = service.totalPages
	} catch (e) {
		error(e)
	} finally {
		loading.value = false
	}
}

async function openCreate() {
	Object.assign(form, {name: '', maxUses: '', expiresAt: '', skipEmailConfirm: true})
	selectedTeams.value = []
	createdUrl.value = ''
	minimumExpiry.value = formatDate(new Date(Date.now() + 60_000), 'YYYY-MM-DD[T]HH:mm')
	createOpen.value = true
	loadingTeams.value = true
	try {
		const result = await teamService.search()
		initialTeams = result.items
		teamResults.value = result.items
		teamCount = result.total
	} catch (e) {
		error(e)
	} finally {
		loadingTeams.value = false
	}
}

const searchTeams = useDebounceFn(async (query: string) => {
	if (teamCount <= initialTeams.length) {
		teamResults.value = initialTeams.filter(team => team.name.toLowerCase().includes(query.toLowerCase()))
		return
	}
	loadingTeams.value = true
	try {
		teamResults.value = (await teamService.search(query)).items
	} catch (e) {
		error(e)
	} finally {
		loadingTeams.value = false
	}
}, 250)

function closeCreate() {
	if (creating.value) return
	createOpen.value = false
	createdUrl.value = ''
}

function selectLink(event: FocusEvent) {
	(event.target as HTMLInputElement).select()
}

async function submitCreate() {
	if (creating.value) return
	const expiresAt = form.expiresAt ? new Date(form.expiresAt) : null
	if (expiresAt && expiresAt.getTime() <= Date.now()) {
		error({message: t('admin.inviteLinks.futureExpiry')})
		return
	}
	creating.value = true
	try {
		const link = await service.createLink({name: form.name, teamIds: selectedTeams.value.map(team => team.id), maxUses: form.maxUses === '' ? null : Number(form.maxUses), expiresAt, skipEmailConfirm: form.skipEmailConfirm})
		const base = configStore.frontendUrl || new URL(import.meta.env.BASE_URL, window.location.origin).toString()
		createdUrl.value = new URL(`invite/${link.token}`, base.endsWith('/') ? base : `${base}/`).toString()
		await loadLinks(1)
	} catch (e) {
		error(e)
	} finally {
		creating.value = false
	}
}

async function deleteLink() {
	if (!pendingDelete.value || deleting.value) return
	deleting.value = true
	try {
		await service.deleteLink(pendingDelete.value.id)
		pendingDelete.value = null
		success({message: t('admin.inviteLinks.deleted')})
		await loadLinks(links.value.length === 1 && page.value > 1 ? page.value - 1 : page.value)
	} catch (e) {
		error(e)
	} finally {
		deleting.value = false
	}
}

onMounted(() => loadLinks())
</script>
