// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package models

import "fmt"

func validateTaskRecurrenceSeries(series *TaskRecurrenceSeries) error {
	if series == nil {
		return nil
	}

	if series.RootTaskID <= 0 {
		return fmt.Errorf("root task is required")
	}

	if series.ProjectID <= 0 {
		return fmt.Errorf("project is required")
	}

	if series.CreatedByID <= 0 {
		return fmt.Errorf("creator is required")
	}

	if series.Interval < 1 {
		return fmt.Errorf("recurrence interval must be at least 1")
	}

	if series.StartDate.IsZero() {
		return fmt.Errorf("recurrence start date is required")
	}

	switch series.Frequency {
	case TaskRecurrenceFrequencyDay,
		TaskRecurrenceFrequencyWeek,
		TaskRecurrenceFrequencyMonth,
		TaskRecurrenceFrequencyYear:
	default:
		return fmt.Errorf("invalid recurrence frequency: %d", series.Frequency)
	}

	switch series.Basis {
	case TaskRecurrenceBasisSchedule,
		TaskRecurrenceBasisCompletion:
	default:
		return fmt.Errorf("invalid recurrence basis: %d", series.Basis)
	}

	switch series.EndType {
	case TaskRecurrenceEndNever:
		if !series.EndDate.IsZero() || series.EndAfterOccurrences != 0 {
			return fmt.Errorf("never-ending recurrence cannot define an end date or occurrence limit")
		}

	case TaskRecurrenceEndDate:
		if series.EndDate.IsZero() {
			return fmt.Errorf("end date is required")
		}
		if series.EndDate.Before(series.StartDate) {
			return fmt.Errorf("end date cannot be before start date")
		}
		if series.EndAfterOccurrences != 0 {
			return fmt.Errorf("date-ended recurrence cannot define an occurrence limit")
		}

	case TaskRecurrenceEndOccurrences:
		if series.EndAfterOccurrences < 1 {
			return fmt.Errorf("occurrence limit must be at least 1")
		}
		if !series.EndDate.IsZero() {
			return fmt.Errorf("occurrence-ended recurrence cannot define an end date")
		}

	default:
		return fmt.Errorf("invalid recurrence end type: %d", series.EndType)
	}

	if series.CreateBeforeDays < 0 {
		return fmt.Errorf("create-before days cannot be negative")
	}

	switch series.WeekendPolicy {
	case TaskRecurrenceWeekendKeep,
		TaskRecurrenceWeekendPreviousBusinessDay,
		TaskRecurrenceWeekendNextBusinessDay:
	default:
		return fmt.Errorf("invalid weekend policy: %d", series.WeekendPolicy)
	}

	switch series.MissedPolicy {
	case TaskRecurrenceMissedNextFuture,
		TaskRecurrenceMissedEveryOccurrence:
	default:
		return fmt.Errorf("invalid missed-occurrence policy: %d", series.MissedPolicy)
	}

	return nil
}
