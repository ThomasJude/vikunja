<template>
	<div class="recurrence-series-editor">
		<div class="editor-header">
			<div>
				<p class="help-text">
					Configure when this task repeats and how future occurrences are created.
				</p>
			</div>

			<span
				v-if="hasSeries"
				class="tag"
				:class="draft.paused ? 'is-warning' : 'is-success'"
			>
				{{ draft.paused ? 'Paused' : 'Active' }}
			</span>
		</div>

		<div
			v-if="errorMessage"
			class="notification is-danger is-light"
		>
			{{ errorMessage }}
		</div>

		<div
			v-if="!dueDate && !hasSeries"
			class="notification is-warning is-light"
		>
			Set a due date for this task first. Recurrence uses it as the initial scheduled occurrence.
		</div>

		<section class="editor-section">
			<h4>Recurrence rule</h4>

			<div class="frequency-buttons">
				<x-button
					v-for="option in frequencyOptions"
					:key="option.value"
					type="button"
					variant="secondary"
					:class="{
						'recurrence-choice-active':
							rule.frequency === option.value,
					}"
					:aria-pressed="rule.frequency === option.value"
					:disabled="disabled || loading"
					@click="selectFrequency(option.value)"
				>
					{{ option.label }}
				</x-button>
			</div>

			<div
				v-if="rule.frequency === TASK_RECURRENCE_FREQUENCIES.DAY"
				class="rule-panel"
			>
				<label class="field-label">
					Every
					<input
						v-model.number="rule.interval"
						class="input recurrence-number"
						type="number"
						min="1"
						:disabled="disabled || loading"
					>
					{{ rule.interval === 1 ? 'day' : 'days' }}
				</label>
			</div>

			<div
				v-else-if="rule.frequency === TASK_RECURRENCE_FREQUENCIES.WEEK"
				class="rule-panel"
			>
				<label class="field-label">
					Every
					<input
						v-model.number="rule.interval"
						class="input recurrence-number"
						type="number"
						min="1"
						:disabled="disabled || loading"
					>
					{{ rule.interval === 1 ? 'week' : 'weeks' }} on
				</label>
				<div class="preset-row">
					<span class="preset-label">Preset:</span>

					<x-button
						type="button"
						variant="secondary"
						:class="{
							'recurrence-choice-active':
								rule.interval === 1 &&
								rule.byWeekdays === 62,
						}"
						:aria-pressed="
							rule.interval === 1 &&
								rule.byWeekdays === 62
						"
						:disabled="disabled || loading"
						@click="applyWeekdaysPreset"
					>
						Weekdays
					</x-button>
				</div>


				<div class="weekday-buttons">
					<x-button
						v-for="weekday in weekdayOptions"
						:key="weekday.bit"
						type="button"
						variant="secondary"
						:class="{
							'recurrence-choice-active':
								hasWeekday(weekday.bit),
						}"
						:aria-pressed="hasWeekday(weekday.bit)"
						:disabled="disabled || loading"
						@click="toggleWeekday(weekday.bit)"
					>
						{{ weekday.short }}
					</x-button>
				</div>
			</div>

			<div
				v-else-if="rule.frequency === TASK_RECURRENCE_FREQUENCIES.MONTH"
				class="rule-panel"
			>
				<label class="field-label">
					Every
					<input
						v-model.number="rule.interval"
						class="input recurrence-number"
						type="number"
						min="1"
						:disabled="disabled || loading"
					>
					{{ rule.interval === 1 ? 'month' : 'months' }}
				</label>
				<div class="preset-row">
					<span class="preset-label">Preset:</span>

					<x-button
						type="button"
						variant="secondary"
						:class="{
							'recurrence-choice-active':
								rule.interval === 3,
						}"
						:aria-pressed="rule.interval === 3"
						:disabled="disabled || loading"
						@click="applyQuarterlyPreset"
					>
						Quarterly
					</x-button>
				</div>


				<div class="rule-choice">
					<label>
						<input
							v-model="monthlyRuleType"
							type="radio"
							value="monthDay"
							:disabled="disabled || loading"
							@change="setMonthlyRuleType('monthDay')"
						>
						On day
					</label>

					<input
						v-if="monthlyRuleType === 'monthDay'"
						v-model.number="rule.byMonthDay"
						class="input recurrence-number"
						type="number"
						min="1"
						max="31"
						:disabled="disabled || loading"
					>
				</div>

				<div class="rule-choice">
					<label>
						<input
							v-model="monthlyRuleType"
							type="radio"
							value="ordinal"
							:disabled="disabled || loading"
							@change="setMonthlyRuleType('ordinal')"
						>
						On the
					</label>

					<template v-if="monthlyRuleType === 'ordinal'">
						<div class="select">
							<select
								v-model.number="rule.bySetPos"
								:disabled="disabled || loading"
							>
								<option
									v-for="ordinal in ordinalOptions"
									:key="ordinal.value"
									:value="ordinal.value"
								>
									{{ ordinal.label }}
								</option>
							</select>
						</div>

						<div class="select">
							<select
								v-model.number="rule.byWeekdays"
								:disabled="disabled || loading"
							>
								<option
									v-for="weekday in weekdayOptions"
									:key="weekday.bit"
									:value="weekday.bit"
								>
									{{ weekday.label }}
								</option>
							</select>
						</div>
					</template>
				</div>


				<div class="advanced-rule-option">
					<label
						v-if="monthlyRuleType === 'monthDay'"
						class="field"
					>
						<span>If the requested day does not exist</span>

						<span class="select">
							<select
								v-model.number="rule.missingPolicy"
								:disabled="disabled || loading"
							>
								<option :value="TASK_RECURRENCE_MISSING_POLICIES.LAST_VALID">
									Use the last day of the month
								</option>
								<option :value="TASK_RECURRENCE_MISSING_POLICIES.SKIP">
									Skip that occurrence
								</option>
							</select>
						</span>
					</label>

					<label
						v-else-if="rule.bySetPos === 5"
						class="field"
					>
						<span>If the selected fifth weekday does not exist</span>

						<span class="select">
							<select
								v-model.number="rule.missingPolicy"
								:disabled="disabled || loading"
							>
								<option :value="TASK_RECURRENCE_MISSING_POLICIES.SKIP">
									Skip that month
								</option>
								<option :value="TASK_RECURRENCE_MISSING_POLICIES.LAST_OCCURRENCE">
									Use the last occurrence of that weekday
								</option>
								<option :value="TASK_RECURRENCE_MISSING_POLICIES.NEXT_PERIOD">
									Use the first occurrence of that weekday in the following month
								</option>
							</select>
						</span>
					</label>
				</div>
			</div>

			<div
				v-else-if="rule.frequency === TASK_RECURRENCE_FREQUENCIES.YEAR"
				class="rule-panel"
			>
				<label class="field-label">
					Every
					<input
						v-model.number="rule.interval"
						class="input recurrence-number"
						type="number"
						min="1"
						:disabled="disabled || loading"
					>
					{{ rule.interval === 1 ? 'year' : 'years' }}
				</label>

				<div class="rule-choice">
					<label>
						<input
							v-model="yearlyRuleType"
							type="radio"
							value="monthDay"
							:disabled="disabled || loading"
							@change="setYearlyRuleType('monthDay')"
						>
						On
					</label>

					<template v-if="yearlyRuleType === 'monthDay'">
						<div class="select">
							<select
								v-model.number="rule.byMonth"
								:disabled="disabled || loading"
							>
								<option
									v-for="month in monthOptions"
									:key="month.value"
									:value="month.value"
								>
									{{ month.label }}
								</option>
							</select>
						</div>

						<input
							v-model.number="rule.byMonthDay"
							class="input recurrence-number"
							type="number"
							min="1"
							max="31"
							:disabled="disabled || loading"
						>
					</template>
				</div>

				<div class="rule-choice">
					<label>
						<input
							v-model="yearlyRuleType"
							type="radio"
							value="ordinal"
							:disabled="disabled || loading"
							@change="setYearlyRuleType('ordinal')"
						>
						On the
					</label>

					<template v-if="yearlyRuleType === 'ordinal'">
						<div class="select">
							<select
								v-model.number="rule.bySetPos"
								:disabled="disabled || loading"
							>
								<option
									v-for="ordinal in ordinalOptions"
									:key="ordinal.value"
									:value="ordinal.value"
								>
									{{ ordinal.label }}
								</option>
							</select>
						</div>

						<div class="select">
							<select
								v-model.number="rule.byWeekdays"
								:disabled="disabled || loading"
							>
								<option
									v-for="weekday in weekdayOptions"
									:key="weekday.bit"
									:value="weekday.bit"
								>
									{{ weekday.label }}
								</option>
							</select>
						</div>

						<span>of</span>

						<div class="select">
							<select
								v-model.number="rule.byMonth"
								:disabled="disabled || loading"
							>
								<option
									v-for="month in monthOptions"
									:key="month.value"
									:value="month.value"
								>
									{{ month.label }}
								</option>
							</select>
						</div>
					</template>
				</div>


				<div class="advanced-rule-option">
					<label
						v-if="yearlyRuleType === 'monthDay'"
						class="field"
					>
						<span>If the requested date does not exist</span>

						<span class="select">
							<select
								v-model.number="rule.missingPolicy"
								:disabled="disabled || loading"
							>
								<option :value="TASK_RECURRENCE_MISSING_POLICIES.LAST_VALID">
									Use the last day of the month
								</option>
								<option :value="TASK_RECURRENCE_MISSING_POLICIES.SKIP">
									Skip that occurrence
								</option>
							</select>
						</span>
					</label>

					<label
						v-else-if="rule.bySetPos === 5"
						class="field"
					>
						<span>If the selected fifth weekday does not exist</span>

						<span class="select">
							<select
								v-model.number="rule.missingPolicy"
								:disabled="disabled || loading"
							>
								<option :value="TASK_RECURRENCE_MISSING_POLICIES.SKIP">
									Skip that occurrence
								</option>
								<option :value="TASK_RECURRENCE_MISSING_POLICIES.LAST_OCCURRENCE">
									Use the last occurrence of that weekday
								</option>
								<option :value="TASK_RECURRENCE_MISSING_POLICIES.NEXT_PERIOD">
									Use the first occurrence of that weekday in the following month
								</option>
							</select>
						</span>
					</label>
				</div>
			</div>

			<div class="rule-start-date">
				<label class="field">
					<span>Start date</span>

					<input
						v-model="startDateInput"
						class="input"
						type="date"
						required
						:disabled="disabled || loading"
					>
				</label>

				<p class="help-text">
					The recurrence begins from this date.
				</p>
			</div>
		</section>

		<section class="editor-section">
			<h4>Create timing</h4>

			<label class="inline-field">
				Create task
				<input
					v-model.number="draft.createBeforeDays"
					class="input recurrence-number"
					type="number"
					min="0"
					:disabled="disabled || loading"
				>
				{{ draft.createBeforeDays === 1 ? 'day' : 'days' }} before due date
			</label>

			<p class="help-text">
				Use 0 to create the task on its due date.
			</p>
		</section>

		<section class="editor-section">
			<h4>Schedule vs. completion</h4>

			<label class="radio-row">
				<input
					v-model.number="rule.basis"
					type="radio"
					:value="TASK_RECURRENCE_BASES.SCHEDULE"
					:disabled="disabled || loading"
				>
				<span>
					<strong>Repeat on Schedule</strong>
					<small>Keep occurrences on the planned calendar schedule.</small>
				</span>
			</label>

			<label class="radio-row">
				<input
					v-model.number="rule.basis"
					type="radio"
					:value="TASK_RECURRENCE_BASES.COMPLETION"
					:disabled="disabled || loading"
				>
				<span>
					<strong>Repeat After Completion</strong>
					<small>Calculate the next occurrence from when the current task is completed.</small>
				</span>
			</label>
		</section>

		<section class="editor-section">
			<h4>End condition</h4>

			<label class="radio-simple">
				<input
					v-model.number="draft.endType"
					type="radio"
					:value="TASK_RECURRENCE_END_TYPES.NEVER"
					:disabled="disabled || loading"
				>
				Never
			</label>

			<label class="radio-simple">
				<input
					v-model.number="draft.endType"
					type="radio"
					:value="TASK_RECURRENCE_END_TYPES.DATE"
					:disabled="disabled || loading"
				>
				On date
			</label>

			<input
				v-if="draft.endType === TASK_RECURRENCE_END_TYPES.DATE"
				v-model="endDateInput"
				class="input"
				type="date"
				:disabled="disabled || loading"
			>

			<label class="radio-simple">
				<input
					v-model.number="draft.endType"
					type="radio"
					:value="TASK_RECURRENCE_END_TYPES.OCCURRENCES"
					:disabled="disabled || loading"
				>
				After
			</label>

			<label
				v-if="draft.endType === TASK_RECURRENCE_END_TYPES.OCCURRENCES"
				class="inline-field"
			>
				<input
					v-model.number="draft.endAfterOccurrences"
					class="input recurrence-number"
					type="number"
					min="1"
					:disabled="disabled || loading"
				>
				occurrences
			</label>
		</section>

		<section class="editor-section">
			<x-button
				type="button"
				variant="secondary"
				:disabled="disabled || loading"
				@click="showMoreOptions = !showMoreOptions"
			>
				{{ showMoreOptions ? 'Hide more options' : 'More options' }}
			</x-button>

			<div
				v-if="showMoreOptions"
				class="more-options"
			>
				<label class="field">
					<span>Weekend handling</span>

					<span class="select">
						<select
							v-model.number="draft.weekendPolicy"
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
					</span>
				</label>

				<label
					v-if="rule.basis === TASK_RECURRENCE_BASES.SCHEDULE"
					class="field"
				>
					<span>Missed occurrences</span>

					<span class="select">
						<select
							v-model.number="draft.missedPolicy"
							:disabled="disabled || loading"
						>
							<option :value="TASK_RECURRENCE_MISSED_POLICIES.NEXT_FUTURE">
								Next valid future occurrence
							</option>
							<option :value="TASK_RECURRENCE_MISSED_POLICIES.EVERY_OCCURRENCE">
								Generate every missed occurrence
							</option>
						</select>
					</span>
				</label>
			</div>
		</section>

		<section class="editor-section summary-section">
			<h4>Summary</h4>
			<p>{{ summary }}</p>
		</section>

		<div class="editor-actions">
			<x-button
				type="button"
				:loading="loading"
				:disabled="disabled || (!dueDate && !hasSeries)"
				@click="save"
			>
				Save recurrence
			</x-button>

			<x-button
				v-if="hasSeries"
				type="button"
				variant="secondary"
				:disabled="disabled || loading"
				@click="togglePaused"
			>
				{{ draft.paused ? 'Resume series' : 'Pause series' }}
			</x-button>

			<x-button
				v-if="hasSeries"
				type="button"
				variant="secondary"
				:disabled="disabled || loading"
				@click="showRemoveRecurrenceModal = true"
			>
				Remove recurrence
			</x-button>
		</div>

		<Modal
			:enabled="showRemoveRecurrenceModal"
			@close="showRemoveRecurrenceModal = false"
			@submit="removeRecurrence"
		>
			<template #header>
				<span>Remove recurrence?</span>
			</template>

			<template #text>
				<p>
					This task and existing occurrences will remain,
					but no new occurrences will be created.
				</p>
			</template>
		</Modal>
	</div>
</template>

<script setup lang="ts">
import {computed, reactive, ref, watch} from 'vue'

import {
	TASK_RECURRENCE_BASES,
	TASK_RECURRENCE_FREQUENCIES,
	TASK_RECURRENCE_MISSING_POLICIES,
	type ITaskRecurrence,
	type TaskRecurrenceFrequency,
} from '@/modelTypes/ITaskRecurrence'

import {
	TASK_RECURRENCE_END_TYPES,
	TASK_RECURRENCE_MISSED_POLICIES,
	TASK_RECURRENCE_WEEKEND_POLICIES,
	type ITaskRecurrenceSeries,
	type ITaskRecurrenceSeriesState,
} from '@/modelTypes/ITaskRecurrenceSeries'

import TaskRecurrenceSeriesService from '@/services/taskRecurrenceSeries'
import {error as notifyError, success} from '@/message'

type RuleType = 'monthDay' | 'ordinal'

const props = defineProps<{
	taskId: number
	dueDate: Date | null
	recurrence: ITaskRecurrence | null
	disabled?: boolean
}>()

const service = new TaskRecurrenceSeriesService()

const loading = ref(false)
const errorMessage = ref('')
const state = ref<ITaskRecurrenceSeriesState | null>(null)

const showMoreOptions = ref(false)
const showRemoveRecurrenceModal = ref(false)
const startDateInput = ref('')
const endDateInput = ref('')

const monthlyRuleType = ref<RuleType>('monthDay')
const yearlyRuleType = ref<RuleType>('monthDay')

const frequencyOptions = [
	{value: TASK_RECURRENCE_FREQUENCIES.DAY, label: 'Daily'},
	{value: TASK_RECURRENCE_FREQUENCIES.WEEK, label: 'Weekly'},
	{value: TASK_RECURRENCE_FREQUENCIES.MONTH, label: 'Monthly'},
	{value: TASK_RECURRENCE_FREQUENCIES.YEAR, label: 'Yearly'},
]

const weekdayOptions = [
	{bit: 2, short: 'Mon', label: 'Monday'},
	{bit: 4, short: 'Tue', label: 'Tuesday'},
	{bit: 8, short: 'Wed', label: 'Wednesday'},
	{bit: 16, short: 'Thu', label: 'Thursday'},
	{bit: 32, short: 'Fri', label: 'Friday'},
	{bit: 64, short: 'Sat', label: 'Saturday'},
	{bit: 1, short: 'Sun', label: 'Sunday'},
]

const ordinalOptions = [
	{value: 1, label: 'First'},
	{value: 2, label: 'Second'},
	{value: 3, label: 'Third'},
	{value: 4, label: 'Fourth'},
	{value: 5, label: 'Fifth'},
	{value: -1, label: 'Last'},
]

const monthOptions = [
	{value: 1, label: 'January'},
	{value: 2, label: 'February'},
	{value: 3, label: 'March'},
	{value: 4, label: 'April'},
	{value: 5, label: 'May'},
	{value: 6, label: 'June'},
	{value: 7, label: 'July'},
	{value: 8, label: 'August'},
	{value: 9, label: 'September'},
	{value: 10, label: 'October'},
	{value: 11, label: 'November'},
	{value: 12, label: 'December'},
]

function dueMonth() {
	return props.dueDate ? props.dueDate.getMonth() + 1 : 1
}

function dueMonthDay() {
	return props.dueDate ? props.dueDate.getDate() : 1
}

function dueWeekdayBit() {
	if (!props.dueDate) {
		return 2
	}

	const jsDay = props.dueDate.getDay()

	return jsDay === 0
		? 1
		: 1 << jsDay
}

function defaultRecurrence(): ITaskRecurrence {
	return {
		frequency: TASK_RECURRENCE_FREQUENCIES.MONTH,
		interval: 1,
		basis: TASK_RECURRENCE_BASES.SCHEDULE,
		byWeekdays: 0,
		byMonth: 0,
		byMonthDay: dueMonthDay(),
		bySetPos: 0,
		missingPolicy: TASK_RECURRENCE_MISSING_POLICIES.LAST_VALID,
	}
}

const rule = reactive<ITaskRecurrence>(defaultRecurrence())

const draft = reactive({
	endType: TASK_RECURRENCE_END_TYPES.NEVER as ITaskRecurrenceSeries['endType'],
	endAfterOccurrences: 0,
	createBeforeDays: 0,
	weekendPolicy: TASK_RECURRENCE_WEEKEND_POLICIES.KEEP as ITaskRecurrenceSeries['weekendPolicy'],
	missedPolicy: TASK_RECURRENCE_MISSED_POLICIES.NEXT_FUTURE as ITaskRecurrenceSeries['missedPolicy'],
	paused: false,
})

const hasSeries = computed(() => Boolean(state.value?.series))

function dateInputFromDate(date: Date | null) {
	if (
		!(date instanceof Date) ||
		Number.isNaN(date.getTime())
	) {
		return ''
	}

	return date.toISOString().slice(0, 10)
}

function localDateInput(date: Date) {
	const year = date.getFullYear()
	const month = String(date.getMonth() + 1).padStart(2, '0')
	const day = String(date.getDate()).padStart(2, '0')

	return `${year}-${month}-${day}`
}

function defaultEndDateInput() {
	const today = localDateInput(new Date())

	if (startDateInput.value && startDateInput.value > today) {
		return startDateInput.value
	}

	return today
}

function startTimestamp(date: string) {
	if (!date) {
		return undefined
	}

	if (
		props.dueDate instanceof Date &&
		!Number.isNaN(props.dueDate.getTime())
	) {
		return `${date}${props.dueDate.toISOString().slice(10)}`
	}

	const existingStartDate = state.value?.series?.startDate

	if (existingStartDate) {
		const parsedStartDate = new Date(existingStartDate)

		if (!Number.isNaN(parsedStartDate.getTime())) {
			return `${date}${parsedStartDate.toISOString().slice(10)}`
		}
	}

	return `${date}T12:00:00.000Z`
}

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


function copyRecurrence(source: ITaskRecurrence) {
	rule.frequency = source.frequency
	rule.interval = source.interval
	rule.basis = source.basis
	rule.byWeekdays = source.byWeekdays
	rule.byMonth = source.byMonth
	rule.byMonthDay = source.byMonthDay
	rule.bySetPos = source.bySetPos

	monthlyRuleType.value = source.byMonthDay > 0
		? 'monthDay'
		: 'ordinal'

	yearlyRuleType.value = source.byMonthDay > 0
		? 'monthDay'
		: 'ordinal'

	if (
		source.missingPolicy === TASK_RECURRENCE_MISSING_POLICIES.DEFAULT &&
		(
			source.frequency === TASK_RECURRENCE_FREQUENCIES.MONTH ||
			source.frequency === TASK_RECURRENCE_FREQUENCIES.YEAR
		)
	) {
		rule.missingPolicy = source.byMonthDay > 0
			? TASK_RECURRENCE_MISSING_POLICIES.LAST_VALID
			: TASK_RECURRENCE_MISSING_POLICIES.SKIP
	} else {
		rule.missingPolicy = source.missingPolicy
	}
}

function resetRecurrenceEditor() {
	copyRecurrence(defaultRecurrence())

	draft.endType = TASK_RECURRENCE_END_TYPES.NEVER
	draft.endAfterOccurrences = 0
	draft.createBeforeDays = 0
	draft.weekendPolicy = TASK_RECURRENCE_WEEKEND_POLICIES.KEEP
	draft.missedPolicy = TASK_RECURRENCE_MISSED_POLICIES.NEXT_FUTURE
	draft.paused = false

	startDateInput.value = dateInputFromDate(props.dueDate)
	endDateInput.value = ''
	showMoreOptions.value = false
}

function loadSeries(series: ITaskRecurrenceSeries) {
	copyRecurrence(recurrenceFromSeries(series))

	draft.endType = series.endType
	draft.endAfterOccurrences = series.endAfterOccurrences
	draft.createBeforeDays = series.createBeforeDays
	draft.weekendPolicy = series.weekendPolicy
	draft.missedPolicy = series.missedPolicy
	draft.paused = series.paused

	startDateInput.value = series.startDate
		? series.startDate.slice(0, 10)
		: dateInputFromDate(props.dueDate)

	endDateInput.value =
		series.endType === TASK_RECURRENCE_END_TYPES.DATE &&
		series.endDate &&
		!series.endDate.startsWith('0001-01-01')
			? series.endDate.slice(0, 10)
			: ''
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
			loadSeries(loaded.series)
			return
		}

		resetRecurrenceEditor()

		if (props.recurrence) {
			copyRecurrence(props.recurrence)
		}
	} catch (error) {
		errorMessage.value = error instanceof Error
			? error.message
			: String(error)
	} finally {
		loading.value = false
	}
}


function selectFrequency(frequency: TaskRecurrenceFrequency) {
	rule.frequency = frequency
	rule.interval = Math.max(1, rule.interval || 1)

	if (frequency === TASK_RECURRENCE_FREQUENCIES.DAY) {
		rule.missingPolicy = TASK_RECURRENCE_MISSING_POLICIES.DEFAULT
		rule.byWeekdays = 0
		rule.byMonth = 0
		rule.byMonthDay = 0
		rule.bySetPos = 0
		return
	}

	if (frequency === TASK_RECURRENCE_FREQUENCIES.WEEK) {
		rule.missingPolicy = TASK_RECURRENCE_MISSING_POLICIES.DEFAULT
		rule.byWeekdays = dueWeekdayBit()
		rule.byMonth = 0
		rule.byMonthDay = 0
		rule.bySetPos = 0
		return
	}

	if (frequency === TASK_RECURRENCE_FREQUENCIES.MONTH) {
		monthlyRuleType.value = 'monthDay'
		rule.missingPolicy = TASK_RECURRENCE_MISSING_POLICIES.LAST_VALID
		rule.byWeekdays = 0
		rule.byMonth = 0
		rule.byMonthDay = dueMonthDay()
		rule.bySetPos = 0
		return
	}

	yearlyRuleType.value = 'monthDay'
	rule.missingPolicy = TASK_RECURRENCE_MISSING_POLICIES.LAST_VALID
	rule.byWeekdays = 0
	rule.byMonth = dueMonth()
	rule.byMonthDay = dueMonthDay()
	rule.bySetPos = 0
}


function setMonthlyRuleType(type: RuleType) {
	monthlyRuleType.value = type

	if (type === 'monthDay') {
		rule.missingPolicy = TASK_RECURRENCE_MISSING_POLICIES.LAST_VALID
		rule.byMonthDay = dueMonthDay()
		rule.byWeekdays = 0
		rule.bySetPos = 0
		return
	}

	rule.missingPolicy = TASK_RECURRENCE_MISSING_POLICIES.SKIP
	rule.byMonthDay = 0
	rule.byWeekdays = dueWeekdayBit()
	rule.bySetPos = 1
}


function setYearlyRuleType(type: RuleType) {
	yearlyRuleType.value = type
	rule.byMonth = rule.byMonth || dueMonth()

	if (type === 'monthDay') {
		rule.missingPolicy = TASK_RECURRENCE_MISSING_POLICIES.LAST_VALID
		rule.byMonthDay = dueMonthDay()
		rule.byWeekdays = 0
		rule.bySetPos = 0
		return
	}

	rule.missingPolicy = TASK_RECURRENCE_MISSING_POLICIES.SKIP
	rule.byMonthDay = 0
	rule.byWeekdays = dueWeekdayBit()
	rule.bySetPos = 1
}

function hasWeekday(bit: number) {
	return (rule.byWeekdays & bit) !== 0
}

function toggleWeekday(bit: number) {
	if (hasWeekday(bit)) {
		rule.byWeekdays &= ~bit
	} else {
		rule.byWeekdays |= bit
	}
}


function applyWeekdaysPreset() {
	rule.interval = 1
	rule.byWeekdays = 2 | 4 | 8 | 16 | 32
}


function applyQuarterlyPreset() {
	rule.interval = 3
}

function ordinalLabel(position: number) {
	return ordinalOptions.find(option => option.value === position)?.label.toLowerCase()
		?? `${position}th`
}

function weekdayLabel(bit: number) {
	return weekdayOptions.find(option => option.bit === bit)?.label
		?? 'selected weekday'
}

function monthLabel(month: number) {
	return monthOptions.find(option => option.value === month)?.label
		?? 'selected month'
}


function dayOfMonthLabel(day: number) {
	const mod100 = day % 100

	if (mod100 >= 11 && mod100 <= 13) {
		return `${day}th`
	}

	switch (day % 10) {
		case 1:
			return `${day}st`
		case 2:
			return `${day}nd`
		case 3:
			return `${day}rd`
		default:
			return `${day}th`
	}
}

function formatList(values: string[]) {
	if (values.length === 0) {
		return 'no weekdays selected'
	}

	if (values.length === 1) {
		return values[0]
	}

	if (values.length === 2) {
		return `${values[0]} and ${values[1]}`
	}

	return `${values.slice(0, -1).join(', ')}, and ${values.at(-1)}`
}

const summary = computed(() => {
	const interval = Math.max(1, rule.interval || 1)

	let text = ''

	switch (rule.frequency) {
		case TASK_RECURRENCE_FREQUENCIES.DAY:
			text = interval === 1
				? 'Every day'
				: `Every ${interval} days`
			break

		case TASK_RECURRENCE_FREQUENCIES.WEEK: {
			if (interval === 1 && rule.byWeekdays === 62) {
				text = 'Every weekday'
				break
			}

			const days = weekdayOptions
				.filter(day => hasWeekday(day.bit))
				.map(day => day.label)

			text = interval === 1
				? `Every week on ${formatList(days)}`
				: `Every ${interval} weeks on ${formatList(days)}`
			break
		}

		case TASK_RECURRENCE_FREQUENCIES.MONTH:
			if (monthlyRuleType.value === 'monthDay') {
				text = interval === 1
					? `Every month on the ${dayOfMonthLabel(rule.byMonthDay)}`
					: `Every ${interval} months on the ${dayOfMonthLabel(rule.byMonthDay)}`

				if (rule.byMonthDay >= 29) {
					if (
						rule.missingPolicy ===
                                                TASK_RECURRENCE_MISSING_POLICIES.LAST_VALID
					) {
						text += ', using the last day of the month when needed'
					} else if (
						rule.missingPolicy ===
                                                TASK_RECURRENCE_MISSING_POLICIES.SKIP
					) {
						text += ', skipping months without that date'
					}
				}
			} else {
				text = interval === 1
					? `The ${ordinalLabel(rule.bySetPos)} ${weekdayLabel(rule.byWeekdays)} of every month`
					: `The ${ordinalLabel(rule.bySetPos)} ${weekdayLabel(rule.byWeekdays)} every ${interval} months`

				if (rule.bySetPos === 5) {
					if (
						rule.missingPolicy ===
                                                TASK_RECURRENCE_MISSING_POLICIES.SKIP
					) {
						text += ', skipping months without one'
					} else if (
						rule.missingPolicy ===
                                                TASK_RECURRENCE_MISSING_POLICIES.LAST_OCCURRENCE
					) {
						text += ', using the last occurrence when a fifth does not exist'
					} else if (
						rule.missingPolicy ===
                                                TASK_RECURRENCE_MISSING_POLICIES.NEXT_PERIOD
					) {
						text += ', using the first occurrence in the following month when needed'
					}
				}
			}
			break

		case TASK_RECURRENCE_FREQUENCIES.YEAR:
			if (yearlyRuleType.value === 'monthDay') {
				text = interval === 1
					? `Every year on ${monthLabel(rule.byMonth)} ${rule.byMonthDay}`
					: `Every ${interval} years on ${monthLabel(rule.byMonth)} ${rule.byMonthDay}`
			} else {
				text = interval === 1
					? `Every year on the ${ordinalLabel(rule.bySetPos)} ${weekdayLabel(rule.byWeekdays)} of ${monthLabel(rule.byMonth)}`
					: `Every ${interval} years on the ${ordinalLabel(rule.bySetPos)} ${weekdayLabel(rule.byWeekdays)} of ${monthLabel(rule.byMonth)}`

				if (
					rule.bySetPos === 5 &&
                                        rule.missingPolicy ===
                                        TASK_RECURRENCE_MISSING_POLICIES.SKIP
				) {
					text += ', skipping years without one'
				}
			}
			break
	}

	if (rule.basis === TASK_RECURRENCE_BASES.COMPLETION) {
		text += ' after completion'
	}

	if (draft.createBeforeDays > 0) {
		const unit = draft.createBeforeDays === 1 ? 'day' : 'days'
		text += `, created ${draft.createBeforeDays} ${unit} before the due date`
	}

	if (
		draft.endType === TASK_RECURRENCE_END_TYPES.DATE &&
                endDateInput.value
	) {
		text += `, ending on ${endDateInput.value}`
	} else if (
		draft.endType === TASK_RECURRENCE_END_TYPES.OCCURRENCES
	) {
		text += `, ending after ${Math.max(1, draft.endAfterOccurrences)} occurrences`
	}

	return `${text}.`
})

function validate() {
	if (!props.taskId) {
		throw new Error('Save the task before configuring recurrence.')
	}

	if (!props.dueDate) {
		throw new Error('Set a due date before saving the recurring series.')
	}

	if (!startDateInput.value) {
		throw new Error('Choose a start date.')
	}

	if (!Number.isInteger(rule.interval) || rule.interval < 1) {
		throw new Error('Repeat interval must be at least 1.')
	}

	if (
		rule.frequency === TASK_RECURRENCE_FREQUENCIES.WEEK &&
		rule.byWeekdays === 0
	) {
		throw new Error('Choose at least one weekday.')
	}

	if (rule.frequency === TASK_RECURRENCE_FREQUENCIES.MONTH) {
		if (
			monthlyRuleType.value === 'monthDay' &&
			(rule.byMonthDay < 1 || rule.byMonthDay > 31)
		) {
			throw new Error('Monthly day must be between 1 and 31.')
		}

		if (
			monthlyRuleType.value === 'ordinal' &&
			(rule.byWeekdays === 0 || rule.bySetPos === 0)
		) {
			throw new Error('Choose an ordinal and weekday.')
		}
	}

	if (rule.frequency === TASK_RECURRENCE_FREQUENCIES.YEAR) {
		if (rule.byMonth < 1 || rule.byMonth > 12) {
			throw new Error('Choose a valid month.')
		}

		if (
			yearlyRuleType.value === 'monthDay' &&
			(rule.byMonthDay < 1 || rule.byMonthDay > 31)
		) {
			throw new Error('Yearly day must be between 1 and 31.')
		}

		if (
			yearlyRuleType.value === 'ordinal' &&
			(rule.byWeekdays === 0 || rule.bySetPos === 0)
		) {
			throw new Error('Choose an ordinal and weekday.')
		}
	}

	if (draft.endType === TASK_RECURRENCE_END_TYPES.DATE) {
		if (!endDateInput.value) {
			throw new Error('Choose an end date.')
		}

		if (
			startDateInput.value &&
			endDateInput.value < startDateInput.value
		) {
			throw new Error('End date cannot be before the start date.')
		}
	}

	if (
		draft.endType === TASK_RECURRENCE_END_TYPES.OCCURRENCES &&
		draft.endAfterOccurrences < 1
	) {
		throw new Error('Occurrence count must be at least 1.')
	}
}

function sanitizedRule(): ITaskRecurrence {
	const clean: ITaskRecurrence = {
		frequency: rule.frequency,
		interval: Math.max(1, rule.interval),
		basis: rule.basis,
		byWeekdays: 0,
		byMonth: 0,
		byMonthDay: 0,
		bySetPos: 0,
		missingPolicy: TASK_RECURRENCE_MISSING_POLICIES.DEFAULT,
	}

	if (rule.frequency === TASK_RECURRENCE_FREQUENCIES.WEEK) {
		clean.byWeekdays = rule.byWeekdays
		return clean
	}

	if (rule.frequency === TASK_RECURRENCE_FREQUENCIES.MONTH) {
		clean.missingPolicy = rule.missingPolicy

		if (monthlyRuleType.value === 'monthDay') {
			clean.byMonthDay = rule.byMonthDay
		} else {
			clean.byWeekdays = rule.byWeekdays
			clean.bySetPos = rule.bySetPos
		}

		return clean
	}

	if (rule.frequency === TASK_RECURRENCE_FREQUENCIES.YEAR) {
		clean.byMonth = rule.byMonth
		clean.missingPolicy = rule.missingPolicy

		if (yearlyRuleType.value === 'monthDay') {
			clean.byMonthDay = rule.byMonthDay
		} else {
			clean.byWeekdays = rule.byWeekdays
			clean.bySetPos = rule.bySetPos
		}
	}

	return clean
}

function buildSeries(): ITaskRecurrenceSeries {
	validate()

	if (rule.basis === TASK_RECURRENCE_BASES.COMPLETION) {
		draft.missedPolicy = TASK_RECURRENCE_MISSED_POLICIES.NEXT_FUTURE
	}

	const series: ITaskRecurrenceSeries = {
		...sanitizedRule(),
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

	if (startDateInput.value) {
		series.startDate = startTimestamp(startDateInput.value)
	}

	if (
		draft.endType === TASK_RECURRENCE_END_TYPES.DATE &&
		endDateInput.value
	) {
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
			loadSeries(saved.series)
		}

		success({message: 'Recurrence saved.'})
	} catch (error) {
		const message = error instanceof Error
			? error.message
			: String(error)

		errorMessage.value = message
		notifyError({message})
	} finally {
		loading.value = false
	}
}

async function removeRecurrence() {
	showRemoveRecurrenceModal.value = false
	loading.value = true
	errorMessage.value = ''

	try {
		await service.removeRecurrence(props.taskId)

		state.value = await service.get(props.taskId)
		resetRecurrenceEditor()

		success({message: 'Recurrence removed.'})
	} catch (error) {
		const message = error instanceof Error
			? error.message
			: String(error)

		errorMessage.value = message
		notifyError({message})
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
			loadSeries(updated.series)
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
	() => draft.endType,
	endType => {
		if (
			endType === TASK_RECURRENCE_END_TYPES.DATE &&
			(
				!endDateInput.value ||
				endDateInput.value.startsWith('0001-01-01')
			)
		) {
			endDateInput.value = defaultEndDateInput()
		}
	},
)

watch(
	() => props.taskId,
	() => {
		void load()
	},
	{immediate: true},
)

watch(
	() => props.recurrence,
	value => {
		if (value && !hasSeries.value) {
			copyRecurrence(value)
		}
	},
	{deep: true},
)

watch(
	() => props.dueDate,
	value => {
		if (!hasSeries.value && value) {
			if (!startDateInput.value) {
				startDateInput.value = dateInputFromDate(value)
			}
		}
	},
)
</script>

<style scoped>
.recurrence-series-editor {
	width: 100%;
	max-width: 52rem;
	padding: 1rem;
	border: 1px solid var(--grey-200);
	border-radius: 0.5rem;
	background: var(--white);
}

.editor-header,
.editor-actions,
.frequency-buttons,
.preset-row,
.weekday-buttons,
.rule-choice,
.inline-field {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	flex-wrap: wrap;
}

.editor-header {
	justify-content: space-between;
}

.editor-header h3,
.editor-section h4 {
	margin: 0;
}

.help-text {
	color: var(--grey-600);
	font-size: 0.875rem;
}

.editor-section {
	padding-block: 1rem;
	border-top: 1px solid var(--grey-200);
}

.editor-section:first-of-type {
	margin-top: 1rem;
}

.frequency-buttons,
.preset-row {
	margin-top: 0.75rem;
}


.frequency-buttons,
.preset-row,
.weekday-buttons,
.editor-actions {
        display: flex;
        align-items: center;
        flex-wrap: wrap;
        gap: 0.5rem;
}

.preset-label {
	font-weight: 600;
}

.rule-panel {
	margin-top: 1rem;
	padding: 1rem;
	border-radius: 0.4rem;
	background: var(--grey-100);
}

.field-label,
.field,
.radio-row,
.radio-simple {
	display: flex;
	gap: 0.5rem;
	margin-block: 0.6rem;
	align-items: center;
}

.radio-row {
	align-items: flex-start;
}

.radio-row span {
	display: flex;
	flex-direction: column;
}

.radio-row small {
	color: var(--grey-600);
}

.recurrence-number {
	width: 6rem;
}

.rule-choice {
	margin-top: 0.75rem;
}

.rule-start-date {
        margin-top: 1rem;
}

.advanced-rule-option {
	margin-top: 1rem;
}

.two-column {
	display: grid;
	grid-template-columns: repeat(2, minmax(0, 1fr));
	gap: 2rem;
}

.more-options {
	display: grid;
	gap: 0.75rem;
	margin-top: 0.5rem;
}

.summary-section {
	background: var(--grey-100);
	padding-inline: 1rem;
	border-radius: 0.4rem;
}

.summary-section p {
	margin-bottom: 0;
}

.editor-actions {
	margin-top: 1rem;
}

@media (max-width: 768px) {
	.two-column {
		grid-template-columns: 1fr;
	}
}

.recurrence-choice-active {
        outline: 2px solid var(--primary);
        outline-offset: 2px;
}

</style>
