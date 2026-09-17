import type {ITaskRecurrence} from '@/modelTypes/ITaskRecurrence'

export const TASK_RECURRENCE_END_TYPES = {
	NEVER: 0,
	DATE: 1,
	OCCURRENCES: 2,
} as const

export type TaskRecurrenceEndType =
	typeof TASK_RECURRENCE_END_TYPES[keyof typeof TASK_RECURRENCE_END_TYPES]

export const TASK_RECURRENCE_WEEKEND_POLICIES = {
	KEEP: 0,
	PREVIOUS_BUSINESS_DAY: 1,
	NEXT_BUSINESS_DAY: 2,
} as const

export type TaskRecurrenceWeekendPolicy =
	typeof TASK_RECURRENCE_WEEKEND_POLICIES[keyof typeof TASK_RECURRENCE_WEEKEND_POLICIES]

export const TASK_RECURRENCE_MISSED_POLICIES = {
	NEXT_FUTURE: 0,
	EVERY_OCCURRENCE: 1,
} as const

export type TaskRecurrenceMissedPolicy =
	typeof TASK_RECURRENCE_MISSED_POLICIES[keyof typeof TASK_RECURRENCE_MISSED_POLICIES]

export const TASK_RECURRENCE_SCOPES = {
	OCCURRENCE: 0,
	THIS_AND_FUTURE: 1,
	ENTIRE_SERIES: 2,
} as const

export type TaskRecurrenceScope =
	typeof TASK_RECURRENCE_SCOPES[keyof typeof TASK_RECURRENCE_SCOPES]

export interface ITaskRecurrenceSeries extends ITaskRecurrence {
	id?: number
	rootTaskId?: number
	projectId?: number
	createdById?: number

	startDate?: string

	endType: TaskRecurrenceEndType
	endDate?: string | null
	endAfterOccurrences: number

	createBeforeDays: number
	weekendPolicy: TaskRecurrenceWeekendPolicy
	missedPolicy: TaskRecurrenceMissedPolicy
	paused: boolean

	created?: string
	updated?: string
}

export interface ITaskRecurrenceOccurrence {
	id: number
	seriesId: number
	taskId: number
	sequence: number
	scheduledDueDate: string
	dueDate: string
	isException: boolean
	exceptionAnchor?: string | null
	created?: string
	updated?: string
}

export interface ITaskRecurrenceSeriesState {
	series: ITaskRecurrenceSeries | null
	occurrence: ITaskRecurrenceOccurrence | null
}
