<template>
	<div class="recurrence-series-options box mbs-3">
		<div class="series-heading">
			<div>
				<strong>Monthly series options</strong>
				<p class="has-text-grey is-size-7">
					These options apply to the recurring series created from this task.
				</p>
			</div>

			<span
				v-if="hasSeries"
				class="tag"
				:class="draft.paused ? 'is-warning' : 'is-success'"
			>
				{{ draft.paused ? "Paused" : "Active" }}
			</span>
		</div>

		<p
			v-if="!dueDate"
			class="notification is-warning is-light"
		>
			Set a due date on this task before saving a recurring series.
		</p>

		<p
			v-if="!isMonthlyRule"
			class="notification is-info is-light"
		>
			Choose <strong>Advanced Schedule → Monthly</strong> above to configure the monthly rule.
		</p>

		<template v-if="isMonthlyRule">
			<div class="series-grid">
				<label>
					<span>End</span>
					<select
						v-model.number="draft.endType"
						class="select-input"
						:disabled="disabled || loading"
						@change="normalizeEnd"
					>
						<option :value="TASK_RECURRENCE_END_TYPES.NEVER">
							Never
						</option>
						<option :value="TASK_RECURRENCE_END_TYPES.DATE">
							On date
						</option>
						<option :value="TASK_RECURRENCE_END_TYPES.OCCURRENCES">
							After occurrences
						</option>
					</select>
				</label>

				<label v-if="draft.endType === TASK_RECURRENCE_END_TYPES.DATE">
					<span>End date</span>
					<input
						v-model="endDateInput"
						type="date"
						class="input"
						:disabled="disabled || loading"
					>
				</label>

				<label v-if="draft.endType === TASK_RECURRENCE_END_TYPES.OCCURRENCES">
					<span>Occurrences</span>
					<input
						v-model.number="draft.endAfterOccurrences"
						type="number"
						min="1"
						class="input"
						:disabled="disabled || loading"
					>
				</label>

				<label>
					<span>Create task</span>
					<div class="inline-number">
						<input
							v-model.number="draft.createBeforeDays"
							type="number"
							min="0"
							class="input"
							:disabled="disabled || loading"
						>
						<span>days before due</span>
					</div>
				</label>

				<label>
					<span>Weekend handling</span>
					<select
						v-model.number="draft.weekendPolicy"
						class="select-input"
						:disabled="disabled || loading"
					>
						<option :value="TASK_RECURRENCE_WEEKEND_POLICIES.KEEP">
							Keep scheduled date
						</option>
						<option :value="TASK_RECURRENCE_WEEKEND_POLICIES.PREVIOUS_BUSINESS_DAY">
							Previous business day
						</option>
						<option :value="TASK_RECURRENCE_WEEKEND_POLICIES.NEXT_BUSINESS_DAY">
							Next business day
						</option>
					</select>
				</label>

				<label v-if="recurrence?.basis === TASK_RECURRENCE_BASES.SCHEDULE">
					<span>Missed occurrences</span>
					<select
						v-model.number="draft.missedPolicy"
						class="select-input"
						:disabled="disabled || loading"
					>
						<option :value="TASK_RECURRENCE_MISSED_POLICIES.NEXT_FUTURE">
							Create next valid future occurrence
						</option>
						<option :value="TASK_RECURRENCE_MISSED_POLICIES.EVERY_OCCURRENCE">
							Create every missed occurrence
						</option>
					</select>
				</label>
			</div>

			<div class="series-summary">
				<strong>Summary</strong>
				<p>{{ summary }}</p>
			</div>

			<p
				v-if="errorMessage"
				class="notification is-danger is-light"
			>
				{{ errorMessage }}
			</p>

			<div class="series-actions">
				<button
					type="button"
					class="button is-primary"
					:class="{'is-loading': loading}"
					:disabled="disabled || loading || !dueDate"
					@click="save"
				>
					{{ hasSeries ? "Save series changes" : "Save monthly schedule" }}
				</button>

				<button
					v-if="hasSeries"
					type="button"
					class="button"
					:class="draft.paused ? 'is-success' : 'is-warning'"
					:disabled="disabled || loading"
					@click="togglePaused"
				>
					{{ draft.paused ? "Resume series" : "Pause series" }}
				</button>
			</div>
		</template>
	</div>
</template>

<script setup lang="ts">
import {computed, reactive, ref, watch} from 'vue'

import type {ITaskRecurrence} from '@/modelTypes/ITaskRecurrence'
import {
	TASK_RECURRENCE_BASES,
	TASK_RECURRENCE_FREQUENCIES,
} from '@/modelTypes/ITaskRecurrence'

import {
	TASK_RECURRENCE_END_TYPES,
	TASK_RECURRENCE_MISSED_POLICIES,
	TASK_RECURRENCE_WEEKEND_POLICIES,
	type ITaskRecurrenceSeries,
	type ITaskRecurrenceSeriesState,
} from '@/modelTypes/ITaskRecurrenceSeries'

import TaskRecurrenceSeriesService from '@/services/taskRecurrenceSeries'

const props = defineProps<{
	taskId: number
	dueDate: Date | null
	recurrence: ITaskRecurrence | null
	disabled?: boolean
}>()

const emit = defineEmits<{
	'update:recurrence': [value: ITaskRecurrence | null]
}>()

const service = new TaskRecurrenceSeriesService()

const loading = ref(false)
const errorMessage = ref('')
const state = ref<ITaskRecurrenceSeriesState | null>(null)
const endDateInput = ref('')

const draft = reactive({
	endType: TASK_RECURRENCE_END_TYPES.NEVER as ITaskRecurrenceSeries['endType'],
	endAfterOccurrences: 0,
	createBeforeDays: 0,
	weekendPolicy: TASK_RECURRENCE_WEEKEND_POLICIES.KEEP as ITaskRecurrenceSeries['weekendPolicy'],
	missedPolicy: TASK_RECURRENCE_MISSED_POLICIES.NEXT_FUTURE as ITaskRecurrenceSeries['missedPolicy'],
	paused: false,
})

const hasSeries = computed(() => state.value?.series !== null && state.value?.series !== undefined)

const isMonthlyRule = computed(() =>
	props.recurrence?.frequency === TASK_RECURRENCE_FREQUENCIES.MONTH,
)

const ordinalNames: Record<number, string> = {
	1: 'first',
	2: 'second',
	3: 'third',
	4: 'fourth',
	5: 'fifth',
	'-1': 'last',
}

const weekdayNames: Record<number, string> = {
	1: 'Sunday',
	2: 'Monday',
	4: 'Tuesday',
	8: 'Wednesday',
	16: 'Thursday',
	32: 'Friday',
	64: 'Saturday',
}

const summary = computed(() => {
	const rule = props.recurrence

	if (!rule || rule.frequency !== TASK_RECURRENCE_FREQUENCIES.MONTH) {
		return ''
	}

	const every = rule.interval === 1
		? 'Every month'
		: `Every ${rule.interval} months`

	let on: string

	if (rule.byMonthDay > 0) {
		on = ` on day ${rule.byMonthDay}`
	} else {
		const ordinal = ordinalNames[rule.bySetPos] ?? `${rule.bySetPos}th`
		const weekday = weekdayNames[rule.byWeekdays] ?? 'selected weekday'
		on = ` on the ${ordinal} ${weekday}`
	}

	const basis = rule.basis === TASK_RECURRENCE_BASES.COMPLETION
		? ', repeating after completion'
		: ', repeating on schedule'

	const createBefore = draft.createBeforeDays > 0
		? `, created ${draft.createBeforeDays} day${draft.createBeforeDays === 1 ? '' : 's'} before due`
		: ', created on its due date'

	let end = ''

	if (draft.endType === TASK_RECURRENCE_END_TYPES.DATE && endDateInput.value) {
		end = `, ending ${endDateInput.value}`
	} else if (draft.endType === TASK_RECURRENCE_END_TYPES.OCCURRENCES) {
		end = `, ending after ${draft.endAfterOccurrences} occurrences`
	}

	return `${every}${on}${basis}${createBefore}${end}.`
})

function recurrenceFromSeries(series: ITaskRecurrenceSeries): ITaskRecurrence {
	return {
		frequency: series.frequency,
		interval: series.interval,
		basis: series.basis,
		byWeekdays: series.byWeekdays,
		byMonth: series.byMonth,
		byMonthDay: series.byMonthDay,
		bySetPos: series.bySetPos,
		missingPolicy: series.missingPolicy,
	}
}

function loadDraft(series: ITaskRecurrenceSeries) {
	draft.endType = series.endType
	draft.endAfterOccurrences = series.endAfterOccurrences
	draft.createBeforeDays = series.createBeforeDays
	draft.weekendPolicy = series.weekendPolicy
	draft.missedPolicy = series.missedPolicy
	draft.paused = series.paused

	endDateInput.value = series.endDate
		? series.endDate.slice(0, 10)
		: ''

	emit('update:recurrence', recurrenceFromSeries(series))
}

async function load() {
	if (!props.taskId) {
		return
	}

	loading.value = true
	errorMessage.value = ''

	try {
		const loaded = await service.get(props.taskId)
		state.value = loaded

		if (loaded.series) {
			loadDraft(loaded.series)
		}
	} catch (error) {
		errorMessage.value = error instanceof Error
			? error.message
			: String(error)
	} finally {
		loading.value = false
	}
}

function normalizeEnd() {
	if (draft.endType === TASK_RECURRENCE_END_TYPES.NEVER) {
		endDateInput.value = ''
		draft.endAfterOccurrences = 0
		return
	}

	if (draft.endType === TASK_RECURRENCE_END_TYPES.DATE) {
		draft.endAfterOccurrences = 0
		return
	}

	endDateInput.value = ''

	if (draft.endAfterOccurrences < 1) {
		draft.endAfterOccurrences = 1
	}
}

function buildSeries(): ITaskRecurrenceSeries {
	if (!props.recurrence) {
		throw new Error('Choose a monthly recurrence rule first.')
	}

	if (props.recurrence.frequency !== TASK_RECURRENCE_FREQUENCIES.MONTH) {
		throw new Error('Only Monthly recurrence is enabled in this editor.')
	}

	if (!props.dueDate) {
		throw new Error('Set a due date before saving the recurring series.')
	}

	if (props.recurrence.basis === TASK_RECURRENCE_BASES.COMPLETION) {
		draft.missedPolicy = TASK_RECURRENCE_MISSED_POLICIES.NEXT_FUTURE
	}

	const series: ITaskRecurrenceSeries = {
		...props.recurrence,

		endType: draft.endType,
		endAfterOccurrences:
			draft.endType === TASK_RECURRENCE_END_TYPES.OCCURRENCES
				? Math.max(1, draft.endAfterOccurrences)
				: 0,

		createBeforeDays: Math.max(0, draft.createBeforeDays),
		weekendPolicy: draft.weekendPolicy,
		missedPolicy: draft.missedPolicy,
		paused: draft.paused,
	}

	if (
		draft.endType === TASK_RECURRENCE_END_TYPES.DATE &&
		endDateInput.value
	) {
		// Keep the requested calendar date stable for America/Chicago. 23:59 UTC
		// is still the same calendar date in Chicago throughout the year.
		series.endDate = `${endDateInput.value}T23:59:59Z`
	}

	return series
}

async function save() {
	loading.value = true
	errorMessage.value = ''

	try {
		const saved = await service.save(
			props.taskId,
			buildSeries(),
		)

		state.value = saved

		if (saved.series) {
			loadDraft(saved.series)
		}
	} catch (error) {
		errorMessage.value = error instanceof Error
			? error.message
			: String(error)
	} finally {
		loading.value = false
	}
}

async function togglePaused() {
	loading.value = true
	errorMessage.value = ''

	try {
		const updated = await service.setPaused(
			props.taskId,
			!draft.paused,
		)

		state.value = updated

		if (updated.series) {
			loadDraft(updated.series)
		}
	} catch (error) {
		errorMessage.value = error instanceof Error
			? error.message
			: String(error)
	} finally {
		loading.value = false
	}
}

watch(
	() => props.taskId,
	() => load(),
	{immediate: true},
)
</script>

<style scoped lang="scss">
.recurrence-series-options {
	margin-block-start: 1rem;
}

.series-heading {
	display: flex;
	align-items: flex-start;
	justify-content: space-between;
	gap: 1rem;
	margin-block-end: 1rem;
}

.series-grid {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
	gap: 1rem;
}

.series-grid label {
	display: flex;
	flex-direction: column;
	gap: .35rem;
	font-weight: 600;
}

.select-input {
	width: 100%;
	min-height: 2.5rem;
	padding: .4rem .65rem;
	border: 1px solid var(--input-border-color);
	border-radius: var(--input-radius);
	background: var(--input-background);
	color: var(--text);
}

.inline-number {
	display: flex;
	align-items: center;
	gap: .5rem;
}

.inline-number .input {
	max-width: 6rem;
}

.series-summary {
	margin-block-start: 1rem;
	padding: .75rem;
	border-radius: .4rem;
	background: var(--grey-100);
}

.series-summary p {
	margin-block-start: .25rem;
}

.series-actions {
	display: flex;
	flex-wrap: wrap;
	gap: .75rem;
	margin-block-start: 1rem;
}
</style>
