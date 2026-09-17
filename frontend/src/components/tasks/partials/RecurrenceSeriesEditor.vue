<template>
	<div class="recurrence-series-editor">
		<div class="editor-header">
			<div>
				<h3>Repeat</h3>
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
			v-if="!dueDate"
			class="notification is-warning is-light"
		>
			Set a due date on this task before saving a recurring series.
		</div>

		<section class="editor-section">
			<h4>Recurrence rule</h4>

			<div class="frequency-buttons">
				<button
					v-for="option in frequencyOptions"
					:key="option.value"
					type="button"
					class="button"
					:class="{'is-primary': rule.frequency === option.value}"
					:disabled="disabled || loading"
					@click="selectFrequency(option.value)"
				>
					{{ option.label }}
				</button>
			</div>

			<div class="preset-row">
				<span class="preset-label">Presets:</span>

				<button
					type="button"
					class="button is-small"
					:disabled="disabled || loading"
					@click="applyWeekdaysPreset"
				>
					Weekdays
				</button>

				<button
					type="button"
					class="button is-small"
					:disabled="disabled || loading"
					@click="applyQuarterlyPreset"
				>
					Quarterly
				</button>
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
					day(s)
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
					week(s) on
				</label>

				<div class="weekday-buttons">
					<button
						v-for="weekday in weekdayOptions"
						:key="weekday.bit"
						type="button"
						class="button is-small"
						:class="{'is-primary': hasWeekday(weekday.bit)}"
						:disabled="disabled || loading"
						@click="toggleWeekday(weekday.bit)"
					>
						{{ weekday.short }}
					</button>
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
					month(s)
				</label>

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
					<label>
						If the requested date does not exist
						<span class="select">
							<select
								v-model.number="rule.missingPolicy"
								:disabled="disabled || loading"
							>
								<template v-if="monthlyRuleType === 'monthDay'">
									<option :value="TASK_RECURRENCE_MISSING_POLICIES.DEFAULT">
										Default behavior
									</option>
									<option :value="TASK_RECURRENCE_MISSING_POLICIES.LAST_VALID">
										Use last valid day
									</option>
									<option :value="TASK_RECURRENCE_MISSING_POLICIES.SKIP">
										Skip that month
									</option>
								</template>

								<template v-else>
									<option :value="TASK_RECURRENCE_MISSING_POLICIES.DEFAULT">
										Default behavior
									</option>
									<option :value="TASK_RECURRENCE_MISSING_POLICIES.SKIP">
										Skip that month
									</option>
									<option :value="TASK_RECURRENCE_MISSING_POLICIES.LAST_OCCURRENCE">
										Use last occurrence
									</option>
									<option :value="TASK_RECURRENCE_MISSING_POLICIES.NEXT_PERIOD">
										Use first occurrence next month
									</option>
								</template>
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
					year(s)
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
					<label>
						If the requested date does not exist
						<span class="select">
							<select
								v-model.number="rule.missingPolicy"
								:disabled="disabled || loading"
							>
								<option :value="TASK_RECURRENCE_MISSING_POLICIES.DEFAULT">
									Default behavior
								</option>
								<option :value="TASK_RECURRENCE_MISSING_POLICIES.LAST_VALID">
									Use last valid date
								</option>
								<option :value="TASK_RECURRENCE_MISSING_POLICIES.SKIP">
									Skip that year
								</option>
								<option
									v-if="yearlyRuleType === 'ordinal'"
									:value="TASK_RECURRENCE_MISSING_POLICIES.LAST_OCCURRENCE"
								>
									Use last occurrence
								</option>
								<option
									v-if="yearlyRuleType === 'ordinal'"
									:value="TASK_RECURRENCE_MISSING_POLICIES.NEXT_PERIOD"
								>
									Use first occurrence next period
								</option>
							</select>
						</span>
					</label>
				</div>
			</div>
		</section>

		<section class="editor-section">
			<h4>Repeat behavior</h4>

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

		<section class="editor-section two-column">
			<div>
				<h4>Start</h4>

				<label class="field">
					<span>Start date</span>
					<input
						v-model="startDateInput"
						class="input"
						type="date"
						:disabled="disabled || loading"
					>
				</label>
			</div>

			<div>
				<h4>Ends</h4>

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
				day(s) before due date
			</label>

			<p class="help-text">
				Use 0 to create the task on its due date.
			</p>
		</section>

		<section class="editor-section">
			<button
				type="button"
				class="button is-text more-options-button"
				@click="showMoreOptions = !showMoreOptions"
			>
				{{ showMoreOptions ? 'Hide more options' : 'More options' }}
			</button>

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
			<button
				type="button"
				class="button is-primary"
				:class="{'is-loading': loading}"
				:disabled="disabled || loading || !dueDate"
				@click="save"
			>
				Save recurrence
			</button>

			<button
				v-if="hasSeries"
				type="button"
				class="button"
				:class="draft.paused ? 'is-success' : 'is-warning'"
				:disabled="disabled || loading"
				@click="togglePaused"
			>
				{{ draft.paused ? 'Resume series' : 'Pause series' }}
			</button>
		</div>
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

type RuleType = 'monthDay' | 'ordinal'

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

const showMoreOptions = ref(false)
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
		byWeekdays: dueWeekdayBit(),
		byMonth: dueMonth(),
		byMonthDay: dueMonthDay(),
		bySetPos: 1,
		missingPolicy: TASK_RECURRENCE_MISSING_POLICIES.DEFAULT,
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
	if (!date) {
		return ''
	}

	return date.toISOString().slice(0, 10)
}

function startTimestamp(date: string) {
	if (!date) {
		return undefined
	}

	const suffix = props.dueDate
		? props.dueDate.toISOString().slice(10)
		: 'T12:00:00.000Z'

	return `${date}${suffix}`
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
	rule.missingPolicy = source.missingPolicy

	monthlyRuleType.value = source.byMonthDay > 0
		? 'monthDay'
		: 'ordinal'

	yearlyRuleType.value = source.byMonthDay > 0
		? 'monthDay'
		: 'ordinal'
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
			loadSeries(loaded.series)
			return
		}

		if (props.recurrence) {
			copyRecurrence(props.recurrence)
		}

		if (!startDateInput.value) {
			startDateInput.value = dateInputFromDate(props.dueDate)
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
	rule.missingPolicy = TASK_RECURRENCE_MISSING_POLICIES.DEFAULT

	if (frequency === TASK_RECURRENCE_FREQUENCIES.DAY) {
		rule.byWeekdays = 0
		rule.byMonth = 0
		rule.byMonthDay = 0
		rule.bySetPos = 0
		return
	}

	if (frequency === TASK_RECURRENCE_FREQUENCIES.WEEK) {
		rule.byWeekdays = dueWeekdayBit()
		rule.byMonth = 0
		rule.byMonthDay = 0
		rule.bySetPos = 0
		return
	}

	if (frequency === TASK_RECURRENCE_FREQUENCIES.MONTH) {
		monthlyRuleType.value = 'monthDay'
		rule.byWeekdays = 0
		rule.byMonth = 0
		rule.byMonthDay = dueMonthDay()
		rule.bySetPos = 0
		return
	}

	yearlyRuleType.value = 'monthDay'
	rule.byWeekdays = 0
	rule.byMonth = dueMonth()
	rule.byMonthDay = dueMonthDay()
	rule.bySetPos = 0
}

function setMonthlyRuleType(type: RuleType) {
	monthlyRuleType.value = type

	if (type === 'monthDay') {
		rule.byMonthDay = dueMonthDay()
		rule.byWeekdays = 0
		rule.bySetPos = 0
		return
	}

	rule.byMonthDay = 0
	rule.byWeekdays = dueWeekdayBit()
	rule.bySetPos = 1
}

function setYearlyRuleType(type: RuleType) {
	yearlyRuleType.value = type
	rule.byMonth = rule.byMonth || dueMonth()

	if (type === 'monthDay') {
		rule.byMonthDay = dueMonthDay()
		rule.byWeekdays = 0
		rule.bySetPos = 0
		return
	}

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
	selectFrequency(TASK_RECURRENCE_FREQUENCIES.WEEK)
	rule.interval = 1
	rule.byWeekdays = 2 | 4 | 8 | 16 | 32
}

function applyQuarterlyPreset() {
	selectFrequency(TASK_RECURRENCE_FREQUENCIES.MONTH)
	rule.interval = 3
	rule.byMonthDay = dueMonthDay()
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
			const days = weekdayOptions
				.filter(day => hasWeekday(day.bit))
				.map(day => day.short)
				.join(', ')

			text = interval === 1
				? `Every week on ${days || 'no weekdays selected'}`
				: `Every ${interval} weeks on ${days || 'no weekdays selected'}`
			break
		}

		case TASK_RECURRENCE_FREQUENCIES.MONTH:
			text = interval === 1
				? 'Every month'
				: `Every ${interval} months`

			if (monthlyRuleType.value === 'monthDay') {
				text += ` on day ${rule.byMonthDay}`
			} else {
				text += ` on the ${ordinalLabel(rule.bySetPos)} ${weekdayLabel(rule.byWeekdays)}`
			}
			break

		case TASK_RECURRENCE_FREQUENCIES.YEAR:
			text = interval === 1
				? 'Every year'
				: `Every ${interval} years`

			if (yearlyRuleType.value === 'monthDay') {
				text += ` on ${monthLabel(rule.byMonth)} ${rule.byMonthDay}`
			} else {
				text += ` on the ${ordinalLabel(rule.bySetPos)} ${weekdayLabel(rule.byWeekdays)} of ${monthLabel(rule.byMonth)}`
			}
			break
	}

	text += rule.basis === TASK_RECURRENCE_BASES.COMPLETION
		? ', repeating after completion'
		: ', repeating on schedule'

	if (draft.createBeforeDays > 0) {
		text += `, created ${draft.createBeforeDays} day(s) before the due date`
	}

	if (draft.endType === TASK_RECURRENCE_END_TYPES.DATE && endDateInput.value) {
		text += `, ending ${endDateInput.value}`
	} else if (draft.endType === TASK_RECURRENCE_END_TYPES.OCCURRENCES) {
		text += `, ending after ${Math.max(1, draft.endAfterOccurrences)} occurrences`
	} else {
		text += ', never ending'
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

	if (
		draft.endType === TASK_RECURRENCE_END_TYPES.DATE &&
		!endDateInput.value
	) {
		throw new Error('Choose an end date.')
	}

	if (
		draft.endType === TASK_RECURRENCE_END_TYPES.OCCURRENCES &&
		draft.endAfterOccurrences < 1
	) {
		throw new Error('Occurrence count must be at least 1.')
	}
}

function buildSeries(): ITaskRecurrenceSeries {
	validate()

	if (rule.basis === TASK_RECURRENCE_BASES.COMPLETION) {
		draft.missedPolicy = TASK_RECURRENCE_MISSED_POLICIES.NEXT_FUTURE
	}

	const series: ITaskRecurrenceSeries = {
		...rule,
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

.advanced-rule-option {
	margin-top: 1rem;
}

.two-column {
	display: grid;
	grid-template-columns: repeat(2, minmax(0, 1fr));
	gap: 2rem;
}

.more-options-button {
	padding-inline: 0;
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
</style>
