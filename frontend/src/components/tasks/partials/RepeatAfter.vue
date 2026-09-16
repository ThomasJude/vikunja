<template>
	<div class="control repeat-after-input">
		<div class="button-group mbs-2">
			<XButton
				variant="secondary"
				class="is-small"
				:class="{'is-active': editorMode === 'simple'}"
				:disabled="disabled || undefined"
				@click="setEditorMode('simple')"
			>
				{{ $t('task.repeat.simple') }}
			</XButton>
			<XButton
				variant="secondary"
				class="is-small"
				:class="{'is-active': editorMode === 'advanced'}"
				:disabled="disabled || undefined"
				@click="setEditorMode('advanced')"
			>
				{{ $t('task.repeat.advancedMonthly') }}
			</XButton>
		</div>

		<template v-if="editorMode === 'simple'">
			<div class="button-group mbs-2">
				<XButton
					variant="secondary"
					class="is-small"
					:disabled="disabled || undefined"
					@click="() => setRepeatAfter(1, 'days')"
				>
					{{ $t('task.repeat.everyDay') }}
				</XButton>
				<XButton
					variant="secondary"
					class="is-small"
					:disabled="disabled || undefined"
					@click="() => setRepeatAfter(1, 'weeks')"
				>
					{{ $t('task.repeat.everyWeek') }}
				</XButton>
				<XButton
					variant="secondary"
					class="is-small"
					:disabled="disabled || undefined"
					@click="() => setRepeatAfter(30, 'days')"
				>
					{{ $t('task.repeat.every30d') }}
				</XButton>
			</div>

			<div class="is-flex is-align-items-center mbe-2">
				<label
					for="repeatMode"
					class="is-fullwidth"
				>
					{{ $t('task.repeat.mode') }}:
				</label>
				<div class="control">
					<div class="select">
						<select
							id="repeatMode"
							v-model="task.repeatMode"
							:disabled="disabled || undefined"
							@change="updateSimpleData"
						>
							<option :value="TASK_REPEAT_MODES.REPEAT_MODE_DEFAULT">
								{{ $t('misc.default') }}
							</option>
							<option :value="TASK_REPEAT_MODES.REPEAT_MODE_MONTH">
								{{ $t('task.repeat.monthly') }}
							</option>
							<option :value="TASK_REPEAT_MODES.REPEAT_MODE_FROM_CURRENT_DATE">
								{{ $t('task.repeat.fromCurrentDate') }}
							</option>
						</select>
					</div>
				</div>
			</div>

			<div
				v-if="task.repeatMode !== TASK_REPEAT_MODES.REPEAT_MODE_MONTH"
				class="is-flex"
			>
				<p class="pis-4">
					{{ $t('task.repeat.each') }}
				</p>
				<div class="field has-addons is-fullwidth">
					<div class="control">
						<input
							v-model.number="repeatAfter.amount"
							:disabled="disabled || undefined"
							class="input"
							:placeholder="$t('task.repeat.specifyAmount')"
							type="number"
							min="0"
							@change="updateSimpleData"
						>
					</div>
					<div class="control">
						<div class="select">
							<select
								v-model="repeatAfter.type"
								:disabled="disabled || undefined"
								@change="updateSimpleData"
							>
								<option value="hours">
									{{ $t('task.repeat.hours') }}
								</option>
								<option value="days">
									{{ $t('task.repeat.days') }}
								</option>
								<option value="weeks">
									{{ $t('task.repeat.weeks') }}
								</option>
							</select>
						</div>
					</div>
				</div>
			</div>
		</template>

		<div
			v-else-if="task.recurrence"
			class="advanced-repeat"
		>
			<div class="repeat-row">
				<label for="recurrenceInterval">
					{{ $t('task.repeat.every') }}
				</label>
				<input
					id="recurrenceInterval"
					v-model.number="task.recurrence.interval"
					class="input recurrence-number"
					type="number"
					min="1"
					:disabled="disabled || undefined"
					@change="updateAdvancedData"
				>
				<span>{{ $t('task.repeat.months') }}</span>
			</div>

			<div class="repeat-row">
				<label for="recurrenceRule">
					{{ $t('task.repeat.rule') }}
				</label>
				<div class="select">
					<select
						id="recurrenceRule"
						v-model="advancedRuleType"
						:disabled="disabled || undefined"
						@change="setAdvancedRuleType"
					>
						<option value="monthDay">
							{{ $t('task.repeat.dayOfMonth') }}
						</option>
						<option value="ordinal">
							{{ $t('task.repeat.ordinalWeekday') }}
						</option>
					</select>
				</div>
			</div>

			<template v-if="advancedRuleType === 'monthDay'">
				<div class="repeat-row">
					<label for="recurrenceMonthDay">
						{{ $t('task.repeat.day') }}
					</label>
					<input
						id="recurrenceMonthDay"
						v-model.number="task.recurrence.byMonthDay"
						class="input recurrence-number"
						type="number"
						min="1"
						max="31"
						:disabled="disabled || undefined"
						@change="updateAdvancedData"
					>
				</div>

				<div class="repeat-row">
					<label for="recurrenceMissingDate">
						{{ $t('task.repeat.ifDateMissing') }}
					</label>
					<div class="select">
						<select
							id="recurrenceMissingDate"
							v-model.number="task.recurrence.missingPolicy"
							:disabled="disabled || undefined"
							@change="updateAdvancedData"
						>
							<option :value="TASK_RECURRENCE_MISSING_POLICIES.LAST_VALID">
								{{ $t('task.repeat.useLastValidDay') }}
							</option>
							<option :value="TASK_RECURRENCE_MISSING_POLICIES.SKIP">
								{{ $t('task.repeat.skipMonth') }}
							</option>
						</select>
					</div>
				</div>
			</template>

			<template v-else>
				<div class="repeat-row">
					<label>{{ $t('task.repeat.onThe') }}</label>

					<div class="select">
						<select
							v-model.number="task.recurrence.bySetPos"
							:disabled="disabled || undefined"
							@change="updateOrdinalPosition"
						>
							<option
								v-for="position in ordinalPositions"
								:key="position.value"
								:value="position.value"
							>
								{{ $t(position.label) }}
							</option>
						</select>
					</div>

					<div class="select">
						<select
							v-model.number="task.recurrence.byWeekdays"
							:disabled="disabled || undefined"
							@change="updateAdvancedData"
						>
							<option
								v-for="weekday in weekdays"
								:key="weekday.value"
								:value="weekday.value"
							>
								{{ $t(weekday.label) }}
							</option>
						</select>
					</div>
				</div>

				<div
					v-if="task.recurrence.bySetPos === 5"
					class="repeat-row"
				>
					<label for="recurrenceMissingOrdinal">
						{{ $t('task.repeat.ifFifthMissing') }}
					</label>
					<div class="select">
						<select
							id="recurrenceMissingOrdinal"
							v-model.number="task.recurrence.missingPolicy"
							:disabled="disabled || undefined"
							@change="updateAdvancedData"
						>
							<option :value="TASK_RECURRENCE_MISSING_POLICIES.SKIP">
								{{ $t('task.repeat.skipMonth') }}
							</option>
							<option :value="TASK_RECURRENCE_MISSING_POLICIES.LAST_OCCURRENCE">
								{{ $t('task.repeat.useLastOccurrence') }}
							</option>
							<option :value="TASK_RECURRENCE_MISSING_POLICIES.NEXT_PERIOD">
								{{ $t('task.repeat.useFirstNextMonth') }}
							</option>
						</select>
					</div>
				</div>
			</template>

			<div class="repeat-row">
				<label for="recurrenceBasis">
					{{ $t('task.repeat.basis') }}
				</label>
				<div class="select">
					<select
						id="recurrenceBasis"
						v-model.number="task.recurrence.basis"
						:disabled="disabled || undefined"
						@change="updateAdvancedData"
					>
						<option :value="TASK_RECURRENCE_BASES.SCHEDULE">
							{{ $t('task.repeat.onSchedule') }}
						</option>
						<option :value="TASK_RECURRENCE_BASES.COMPLETION">
							{{ $t('task.repeat.afterCompletion') }}
						</option>
					</select>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup lang="ts">
import {reactive, ref, watch} from 'vue'
import {useI18n} from 'vue-i18n'

import {error} from '@/message'

import {TASK_REPEAT_MODES} from '@/types/IRepeatMode'
import type {IRepeatAfter} from '@/types/IRepeatAfter'
import type {ITask} from '@/modelTypes/ITask'
import {
	TASK_RECURRENCE_BASES,
	TASK_RECURRENCE_FREQUENCIES,
	TASK_RECURRENCE_MISSING_POLICIES,
	type ITaskRecurrence,
} from '@/modelTypes/ITaskRecurrence'
import TaskModel from '@/models/task'

type EditorMode = 'simple' | 'advanced'
type AdvancedRuleType = 'monthDay' | 'ordinal'

const props = withDefaults(defineProps<{
	modelValue: ITask | undefined,
	disabled?: boolean
}>(), {
	disabled: false,
})

const emit = defineEmits<{
	'update:modelValue': [value: ITask | undefined],
}>()

const {t} = useI18n({useScope: 'global'})

const task = ref<ITask>(new TaskModel())
const editorMode = ref<EditorMode>('simple')
const advancedRuleType = ref<AdvancedRuleType>('monthDay')

const repeatAfter = reactive({
	amount: 0,
	type: '',
})

const ordinalPositions = [
	{value: 1, label: 'task.repeat.first'},
	{value: 2, label: 'task.repeat.second'},
	{value: 3, label: 'task.repeat.third'},
	{value: 4, label: 'task.repeat.fourth'},
	{value: 5, label: 'task.repeat.fifth'},
	{value: -1, label: 'task.repeat.last'},
]

const weekdays = [
	{value: 1 << 1, label: 'task.repeat.monday'},
	{value: 1 << 2, label: 'task.repeat.tuesday'},
	{value: 1 << 3, label: 'task.repeat.wednesday'},
	{value: 1 << 4, label: 'task.repeat.thursday'},
	{value: 1 << 5, label: 'task.repeat.friday'},
	{value: 1 << 6, label: 'task.repeat.saturday'},
	{value: 1, label: 'task.repeat.sunday'},
]

watch(
	() => props.modelValue,
	(value: ITask | undefined) => {
		if (!value) {
			return
		}

		task.value = value

		if (typeof value.repeatAfter !== 'undefined') {
			Object.assign(repeatAfter, value.repeatAfter)
		}

		editorMode.value = value.recurrence ? 'advanced' : 'simple'

		if (value.recurrence) {
			advancedRuleType.value = value.recurrence.byMonthDay > 0
				? 'monthDay'
				: 'ordinal'
		}
	},
	{
		immediate: true,
		deep: true,
	},
)

function defaultMonthDay(): number {
	const dueDate = task.value.dueDate

	if (dueDate instanceof Date) {
		return dueDate.getDate()
	}

	return 1
}

function createDefaultRecurrence(): ITaskRecurrence {
	return {
		frequency: TASK_RECURRENCE_FREQUENCIES.MONTH,
		interval: 1,
		basis: TASK_RECURRENCE_BASES.SCHEDULE,
		byWeekdays: 0,
		byMonth: 0,
		byMonthDay: defaultMonthDay(),
		bySetPos: 0,
		missingPolicy: TASK_RECURRENCE_MISSING_POLICIES.LAST_VALID,
	}
}

function setEditorMode(mode: EditorMode) {
	editorMode.value = mode

	if (mode === 'advanced') {
		if (!task.value.recurrence) {
			task.value.recurrence = createDefaultRecurrence()
		}

		task.value.repeatMode = TASK_REPEAT_MODES.REPEAT_MODE_DEFAULT
		Object.assign(task.value.repeatAfter, {
			amount: 0,
			type: 'days',
		})

		advancedRuleType.value = task.value.recurrence.byMonthDay > 0
			? 'monthDay'
			: 'ordinal'
	} else {
		task.value.recurrence = null
	}

	emit('update:modelValue', task.value)
}

function updateSimpleData() {
	if (!task.value ||
		(task.value.repeatMode === TASK_REPEAT_MODES.REPEAT_MODE_DEFAULT && repeatAfter.amount === 0) ||
		(task.value.repeatMode === TASK_REPEAT_MODES.REPEAT_MODE_FROM_CURRENT_DATE && repeatAfter.amount === 0)
	) {
		return
	}

	if (task.value.repeatMode === TASK_REPEAT_MODES.REPEAT_MODE_DEFAULT && repeatAfter.amount < 0) {
		error({message: t('task.repeat.invalidAmount')})
		return
	}

	task.value.recurrence = null
	Object.assign(task.value.repeatAfter, repeatAfter)
	emit('update:modelValue', task.value)
}

function setRepeatAfter(amount: number, type: IRepeatAfter['type']) {
	Object.assign(repeatAfter, {amount, type})
	updateSimpleData()
}

function setAdvancedRuleType() {
	const recurrence = task.value.recurrence
	if (!recurrence) {
		return
	}

	if (advancedRuleType.value === 'monthDay') {
		recurrence.byMonthDay = defaultMonthDay()
		recurrence.byWeekdays = 0
		recurrence.bySetPos = 0
		recurrence.missingPolicy = TASK_RECURRENCE_MISSING_POLICIES.LAST_VALID
	} else {
		recurrence.byMonthDay = 0
		recurrence.byWeekdays = 1 << 1
		recurrence.bySetPos = 1
		recurrence.missingPolicy = TASK_RECURRENCE_MISSING_POLICIES.DEFAULT
	}

	updateAdvancedData()
}

function updateOrdinalPosition() {
	const recurrence = task.value.recurrence
	if (!recurrence) {
		return
	}

	if (recurrence.bySetPos === 5) {
		if (
			recurrence.missingPolicy !== TASK_RECURRENCE_MISSING_POLICIES.SKIP &&
			recurrence.missingPolicy !== TASK_RECURRENCE_MISSING_POLICIES.LAST_OCCURRENCE &&
			recurrence.missingPolicy !== TASK_RECURRENCE_MISSING_POLICIES.NEXT_PERIOD
		) {
			recurrence.missingPolicy = TASK_RECURRENCE_MISSING_POLICIES.SKIP
		}
	} else {
		recurrence.missingPolicy = TASK_RECURRENCE_MISSING_POLICIES.DEFAULT
	}

	updateAdvancedData()
}

function updateAdvancedData() {
	const recurrence = task.value.recurrence
	if (!recurrence) {
		return
	}

	if (recurrence.interval < 1) {
		error({message: t('task.repeat.invalidInterval')})
		return
	}

	if (
		advancedRuleType.value === 'monthDay' &&
		(recurrence.byMonthDay < 1 || recurrence.byMonthDay > 31)
	) {
		error({message: t('task.repeat.invalidMonthDay')})
		return
	}

	recurrence.frequency = TASK_RECURRENCE_FREQUENCIES.MONTH
	task.value.repeatMode = TASK_REPEAT_MODES.REPEAT_MODE_DEFAULT

	Object.assign(task.value.repeatAfter, {
		amount: 0,
		type: 'days',
	})

	emit('update:modelValue', task.value)
}
</script>

<style lang="scss" scoped>
p {
	padding-block-start: 6px;
}

.input {
	min-inline-size: 2rem;
}

.recurrence-number {
	inline-size: 5rem;
}

.repeat-row {
	display: flex;
	align-items: center;
	gap: .5rem;
	margin-block-end: .75rem;

	label {
		min-inline-size: 8rem;
	}
}

.advanced-repeat {
	margin-block-start: 1rem;
}

.button-group {
	display: flex;
	justify-content: center;

	:deep(.button:not(:last-child)) {
		border-start-end-radius: 0;
		border-end-end-radius: 0;
	}

	:deep(.button:not(:first-child)) {
		border-start-start-radius: 0;
		border-end-start-radius: 0;
	}

	:deep(.button.is-active) {
		font-weight: 700;
	}
}
</style>
