<template>
	<Modal
		:enabled="enabled"
		@close="$emit('close')"
		@submit="submit"
	>
		<template #header>
			<span>
				{{ action === 'delete' ? 'Delete recurring task' : 'Edit recurring task' }}
			</span>
		</template>

		<template #text>
			<p class="scope-intro">
				{{ action === 'delete'
					? 'Choose which occurrences should be deleted.'
					: 'Choose which occurrences should receive this change.'
				}}
			</p>

			<div class="recurrence-scope-options">
				<label class="recurrence-scope-option">
					<input
						v-model.number="selectedScope"
						type="radio"
						:value="TASK_RECURRENCE_SCOPES.OCCURRENCE"
					>

					<span>
						<strong>This occurrence only</strong>
						<small>
							Only this task is changed. The recurring series continues normally.
						</small>
					</span>
				</label>

				<label
					class="recurrence-scope-option"
					:class="{'is-disabled': occurrenceOnly}"
				>
					<input
						v-model.number="selectedScope"
						type="radio"
						:value="TASK_RECURRENCE_SCOPES.THIS_AND_FUTURE"
						:disabled="occurrenceOnly"
					>

					<span>
						<strong>This and future occurrences</strong>
						<small>
							Apply the change from this occurrence forward.
						</small>
					</span>
				</label>

				<label
					class="recurrence-scope-option"
					:class="{'is-disabled': occurrenceOnly}"
				>
					<input
						v-model.number="selectedScope"
						type="radio"
						:value="TASK_RECURRENCE_SCOPES.ENTIRE_SERIES"
						:disabled="occurrenceOnly"
					>

					<span>
						<strong>Entire series</strong>
						<small>
							Apply the change to the whole recurring series.
						</small>
					</span>
				</label>
			</div>

			<div
				v-if="occurrenceOnly"
				class="notification is-info is-light recurrence-scope-note"
			>
				Dates belong to an individual occurrence. To change the schedule
				for future tasks, edit the recurrence rule instead.
			</div>

			<div
				v-if="action === 'delete' && selectedScope === TASK_RECURRENCE_SCOPES.ENTIRE_SERIES"
				class="notification is-warning is-light recurrence-scope-note"
			>
				This will delete all materialized occurrences and stop future
				occurrences from being generated.
			</div>
		</template>
	</Modal>
</template>

<script setup lang="ts">
import {ref, watch} from 'vue'

import {
	TASK_RECURRENCE_SCOPES,
	type TaskRecurrenceScope,
} from '@/modelTypes/ITaskRecurrenceSeries'

const props = defineProps<{
	enabled: boolean
	action: 'edit' | 'delete'
	occurrenceOnly?: boolean
}>()

const emit = defineEmits<{
	close: []
	submit: [scope: TaskRecurrenceScope]
}>()

const selectedScope = ref<TaskRecurrenceScope>(
	TASK_RECURRENCE_SCOPES.OCCURRENCE,
)

watch(
	() => props.enabled,
	enabled => {
		if (enabled) {
			selectedScope.value = TASK_RECURRENCE_SCOPES.OCCURRENCE
		}
	},
)

function submit() {
	emit('submit', selectedScope.value)
}
</script>

<style scoped>
.scope-intro {
	margin-block-end: 1rem;
}

.recurrence-scope-options {
	display: grid;
	gap: 0.75rem;
}

.recurrence-scope-option {
	display: flex;
	align-items: flex-start;
	gap: 0.75rem;
	padding: 0.85rem;
	border: 1px solid var(--grey-200);
	border-radius: 0.4rem;
	background: var(--scheme-main);
	color: var(--text);
	cursor: pointer;
}

.recurrence-scope-option span {
	display: flex;
	flex-direction: column;
	gap: 0.15rem;
}

.recurrence-scope-option small {
	color: var(--grey-600);
}

.recurrence-scope-option.is-disabled {
	opacity: 0.55;
	cursor: not-allowed;
}

.recurrence-scope-note {
	margin-block-start: 1rem;
}
</style>
