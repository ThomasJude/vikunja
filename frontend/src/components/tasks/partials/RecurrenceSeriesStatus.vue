<template>
	<BaseButton
		v-if="series"
		class="recurrence-series-status"
		@click="emit('open')"
	>
		<span class="recurrence-series-status-heading">
			<Icon icon="history" />

			<span class="recurrence-series-status-title">
				Repeat
			</span>

			<span
				class="tag"
				:class="series.paused ? 'is-warning' : 'is-success'"
			>
				{{ series.paused ? 'Paused' : 'Active' }}
			</span>
		</span>

		<span class="recurrence-series-status-summary">
			{{ summary }}
		</span>
	</BaseButton>
</template>

<script setup lang="ts">
import {computed, ref, watch} from 'vue'

import TaskRecurrenceSeriesService from '@/services/taskRecurrenceSeries'

import {
	TASK_RECURRENCE_BASES,
	TASK_RECURRENCE_FREQUENCIES,
	TASK_RECURRENCE_MISSING_POLICIES,
} from '@/modelTypes/ITaskRecurrence'

import type {
	ITaskRecurrenceSeries,
	ITaskRecurrenceSeriesState,
} from '@/modelTypes/ITaskRecurrenceSeries'

const props = defineProps<{
	taskId: number
}>()

const emit = defineEmits<{
	open: []
}>()

const service = new TaskRecurrenceSeriesService()

const state = ref<ITaskRecurrenceSeriesState | null>(null)

const series = computed<ITaskRecurrenceSeries | null>(
	() => state.value?.series ?? null,
)

const weekdays = [
	{bit: 2, short: 'Mon', name: 'Monday'},
	{bit: 4, short: 'Tue', name: 'Tuesday'},
	{bit: 8, short: 'Wed', name: 'Wednesday'},
	{bit: 16, short: 'Thu', name: 'Thursday'},
	{bit: 32, short: 'Fri', name: 'Friday'},
	{bit: 64, short: 'Sat', name: 'Saturday'},
	{bit: 1, short: 'Sun', name: 'Sunday'},
]

const months = [
	'',
	'January',
	'February',
	'March',
	'April',
	'May',
	'June',
	'July',
	'August',
	'September',
	'October',
	'November',
	'December',
]

function ordinalNumber(value: number) {
	const mod100 = value % 100

	if (mod100 >= 11 && mod100 <= 13) {
		return `${value}th`
	}

	switch (value % 10) {
		case 1:
			return `${value}st`
		case 2:
			return `${value}nd`
		case 3:
			return `${value}rd`
		default:
			return `${value}th`
	}
}

function ordinalPosition(value: number) {
	switch (value) {
		case 1:
			return 'first'
		case 2:
			return 'second'
		case 3:
			return 'third'
		case 4:
			return 'fourth'
		case 5:
			return 'fifth'
		case -1:
			return 'last'
		default:
			return `${value}th`
	}
}

function weekdayName(mask: number) {
	return weekdays.find(day => day.bit === mask)?.name ?? 'selected weekday'
}

const summary = computed(() => {
	const rule = series.value

	if (!rule) {
		return ''
	}

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

			const selected = weekdays
				.filter(day => (rule.byWeekdays & day.bit) !== 0)
				.map(day => day.name)

			const dayText = selected.length > 1
				? `${selected.slice(0, -1).join(', ')} and ${selected.at(-1)}`
				: selected[0] ?? 'selected weekdays'

			text = interval === 1
				? `Every week on ${dayText}`
				: `Every ${interval} weeks on ${dayText}`
			break
		}

		case TASK_RECURRENCE_FREQUENCIES.MONTH:
			text = interval === 1
				? 'Every month'
				: `Every ${interval} months`

			if (rule.byMonthDay > 0) {
				text += ` on the ${ordinalNumber(rule.byMonthDay)}`

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
				text += ` on the ${ordinalPosition(rule.bySetPos)} ${weekdayName(rule.byWeekdays)}`

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
			text = interval === 1
				? 'Every year'
				: `Every ${interval} years`

			if (rule.byMonthDay > 0) {
				text += ` on ${months[rule.byMonth]} ${rule.byMonthDay}`

				if (
					(rule.byMonth === 2 && rule.byMonthDay > 28) ||
					([4, 6, 9, 11].includes(rule.byMonth) && rule.byMonthDay > 30)
				) {
					if (
						rule.missingPolicy ===
							TASK_RECURRENCE_MISSING_POLICIES.LAST_VALID
					) {
						text += ', using the last day of the month when needed'
					} else if (
						rule.missingPolicy ===
							TASK_RECURRENCE_MISSING_POLICIES.SKIP
					) {
						text += ', skipping years without that date'
					}
				}
			} else {
				text += ` on the ${ordinalPosition(rule.bySetPos)} ${weekdayName(rule.byWeekdays)} of ${months[rule.byMonth]}`

				if (rule.bySetPos === 5) {
					if (
						rule.missingPolicy ===
							TASK_RECURRENCE_MISSING_POLICIES.SKIP
					) {
						text += ', skipping years without one'
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
	}

	if (rule.basis === TASK_RECURRENCE_BASES.COMPLETION) {
		text += ' after completion'
	}

	return `${text}.`
})

async function load() {
	if (!props.taskId) {
		state.value = null
		return
	}

	try {
		state.value = await service.get(props.taskId)
	} catch {
		state.value = null
	}
}

watch(
	() => props.taskId,
	() => {
		void load()
	},
	{immediate: true},
)
</script>

<style scoped>
.recurrence-series-status {
	display: flex;
	flex-direction: column;
	align-items: flex-start;
	gap: 0.25rem;
	width: 100%;
	text-align: left;
}

.recurrence-series-status-heading {
	display: flex;
	align-items: center;
	gap: 0.5rem;
}

.recurrence-series-status-title {
	font-weight: 500;
}

.recurrence-series-status-summary {
	margin-inline-start: 1.75rem;
	color: var(--grey-700);
	font-size: 0.9rem;
}
</style>
