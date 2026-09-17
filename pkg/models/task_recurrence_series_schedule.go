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

import (
	"fmt"
	"time"
)

func taskRecurrenceRuleFromSeries(series *TaskRecurrenceSeries) (*TaskRecurrence, error) {
	if err := validateTaskRecurrenceSeries(series); err != nil {
		return nil, err
	}

	return &TaskRecurrence{
		Frequency:     series.Frequency,
		Interval:      series.Interval,
		Basis:         series.Basis,
		ByWeekdays:    series.ByWeekdays,
		ByMonth:       series.ByMonth,
		ByMonthDay:    series.ByMonthDay,
		BySetPos:      series.BySetPos,
		MissingPolicy: series.MissingPolicy,
	}, nil
}

func taskRecurrenceSeriesDueDate(
	series *TaskRecurrenceSeries,
	scheduled time.Time,
) (time.Time, error) {
	if series == nil {
		return time.Time{}, fmt.Errorf("recurrence series is required")
	}
	if scheduled.IsZero() {
		return time.Time{}, fmt.Errorf("scheduled due date is required")
	}

	switch series.WeekendPolicy {
	case TaskRecurrenceWeekendKeep:
		return scheduled, nil

	case TaskRecurrenceWeekendPreviousBusinessDay:
		switch scheduled.Weekday() {
		case time.Saturday:
			return scheduled.AddDate(0, 0, -1), nil
		case time.Sunday:
			return scheduled.AddDate(0, 0, -2), nil
		default:
			return scheduled, nil
		}

	case TaskRecurrenceWeekendNextBusinessDay:
		switch scheduled.Weekday() {
		case time.Saturday:
			return scheduled.AddDate(0, 0, 2), nil
		case time.Sunday:
			return scheduled.AddDate(0, 0, 1), nil
		default:
			return scheduled, nil
		}

	default:
		return time.Time{}, fmt.Errorf(
			"invalid weekend policy: %d",
			series.WeekendPolicy,
		)
	}
}

func taskRecurrenceSeriesCreateAt(
	series *TaskRecurrenceSeries,
	due time.Time,
) (time.Time, error) {
	if series == nil {
		return time.Time{}, fmt.Errorf("recurrence series is required")
	}
	if due.IsZero() {
		return time.Time{}, fmt.Errorf("due date is required")
	}
	if series.CreateBeforeDays < 0 {
		return time.Time{}, fmt.Errorf("create-before days cannot be negative")
	}

	return due.AddDate(0, 0, -series.CreateBeforeDays), nil
}

func taskRecurrenceSeriesAllowsOccurrence(
	series *TaskRecurrenceSeries,
	sequence int,
	scheduled time.Time,
) (bool, error) {
	if series == nil {
		return false, fmt.Errorf("recurrence series is required")
	}
	if sequence < 1 {
		return false, fmt.Errorf("occurrence sequence must be at least 1")
	}
	if scheduled.IsZero() {
		return false, fmt.Errorf("scheduled due date is required")
	}

	switch series.EndType {
	case TaskRecurrenceEndNever:
		return true, nil

	case TaskRecurrenceEndDate:
		return !scheduled.After(series.EndDate), nil

	case TaskRecurrenceEndOccurrences:
		return sequence <= series.EndAfterOccurrences, nil

	default:
		return false, fmt.Errorf(
			"invalid recurrence end type: %d",
			series.EndType,
		)
	}
}
