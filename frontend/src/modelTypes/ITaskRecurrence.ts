export const TASK_RECURRENCE_FREQUENCIES = {
	DAY: 1,
	WEEK: 2,
	MONTH: 3,
	YEAR: 4,
} as const

export type TaskRecurrenceFrequency =
	typeof TASK_RECURRENCE_FREQUENCIES[keyof typeof TASK_RECURRENCE_FREQUENCIES]

export const TASK_RECURRENCE_BASES = {
	SCHEDULE: 0,
	COMPLETION: 1,
} as const

export type TaskRecurrenceBasis =
	typeof TASK_RECURRENCE_BASES[keyof typeof TASK_RECURRENCE_BASES]

export const TASK_RECURRENCE_MISSING_POLICIES = {
	DEFAULT: 0,
	LAST_VALID: 1,
	SKIP: 2,
	LAST_OCCURRENCE: 3,
	NEXT_PERIOD: 4,
} as const

export type TaskRecurrenceMissingPolicy =
	typeof TASK_RECURRENCE_MISSING_POLICIES[keyof typeof TASK_RECURRENCE_MISSING_POLICIES]

export interface ITaskRecurrence {
	id?: number
	taskId?: number

	frequency: TaskRecurrenceFrequency
	interval: number
	basis: TaskRecurrenceBasis

	byWeekdays: number
	byMonth: number
	byMonthDay: number
	bySetPos: number

	missingPolicy: TaskRecurrenceMissingPolicy

	created?: string
	updated?: string
}
