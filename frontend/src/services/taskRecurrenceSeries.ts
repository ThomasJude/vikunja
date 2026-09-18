import {apiV2Url, AuthenticatedHTTPFactory} from '@/helpers/fetcher'
import {objectToCamelCase, objectToSnakeCase} from '@/helpers/case'

import type {ITask} from '@/modelTypes/ITask'
import type {
	ITaskRecurrenceSeries,
	ITaskRecurrenceSeriesState,
	TaskRecurrenceScope,
} from '@/modelTypes/ITaskRecurrenceSeries'

type SeriesSavePayload = Pick<
	ITaskRecurrenceSeries,
	| 'frequency'
	| 'interval'
	| 'basis'
	| 'byWeekdays'
	| 'byMonth'
	| 'byMonthDay'
	| 'bySetPos'
	| 'missingPolicy'
	| 'endType'
	| 'endAfterOccurrences'
	| 'createBeforeDays'
	| 'weekendPolicy'
	| 'missedPolicy'
	| 'paused'
> & {
	startDate?: string
	endDate?: string
}

export default class TaskRecurrenceSeriesService {
	private readonly http = AuthenticatedHTTPFactory()

	async get(taskId: number): Promise<ITaskRecurrenceSeriesState> {
		const {data} = await this.http.get(
			apiV2Url(`tasks/${taskId}/recurrence-series`),
		)

		return objectToCamelCase(data) as ITaskRecurrenceSeriesState
	}

	async save(
		taskId: number,
		series: ITaskRecurrenceSeries,
	): Promise<ITaskRecurrenceSeriesState> {
		const payload: SeriesSavePayload = {
			frequency: series.frequency,
			interval: series.interval,
			basis: series.basis,

			byWeekdays: series.byWeekdays,
			byMonth: series.byMonth,
			byMonthDay: series.byMonthDay,
			bySetPos: series.bySetPos,
			missingPolicy: series.missingPolicy,

			endType: series.endType,
			endAfterOccurrences: series.endAfterOccurrences,

			createBeforeDays: series.createBeforeDays,
			weekendPolicy: series.weekendPolicy,
			missedPolicy: series.missedPolicy,
			paused: series.paused,
		}

		if (series.startDate) {
			payload.startDate = series.startDate
		}

		if (series.endDate) {
			payload.endDate = series.endDate
		}

		const {data} = await this.http.put(
			apiV2Url(`tasks/${taskId}/recurrence-series`),
			objectToSnakeCase(payload),
		)

		return objectToCamelCase(data) as ITaskRecurrenceSeriesState
	}

	async setPaused(
		taskId: number,
		paused: boolean,
	): Promise<ITaskRecurrenceSeriesState> {
		const {data} = await this.http.patch(
			apiV2Url(`tasks/${taskId}/recurrence-series/pause`),
			{paused},
		)

		return objectToCamelCase(data) as ITaskRecurrenceSeriesState
	}

	async removeRecurrence(taskId: number): Promise<void> {
		await this.http.delete(
			apiV2Url(`tasks/${taskId}/recurrence-series/rule`),
		)
	}

	async scopedUpdate(
		taskId: number,
		scope: TaskRecurrenceScope,
		fields: string[],
		task: Partial<ITask>,
	): Promise<ITaskRecurrenceSeriesState> {
		const {data} = await this.http.patch(
			apiV2Url(`tasks/${taskId}/recurrence-series/task`),
			objectToSnakeCase({
				scope,
				fields,
				task,
			}),
		)

		return objectToCamelCase(data) as ITaskRecurrenceSeriesState
	}

	async deleteScoped(
		taskId: number,
		scope: TaskRecurrenceScope,
	): Promise<void> {
		await this.http.delete(
			apiV2Url(`tasks/${taskId}/recurrence-series`),
			{params: {scope}},
		)
	}
}
