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

type TaskRecurrenceSeriesOccurrencePlan struct {
	Sequence int

	ScheduledDueDate time.Time
	DueDate          time.Time
	CreateAt         time.Time
}

func planNextTaskRecurrenceSeriesOccurrence(
	series *TaskRecurrenceSeries,
	current *TaskRecurrenceOccurrence,
	reference time.Time,
) (*TaskRecurrenceSeriesOccurrencePlan, error) {
	if series == nil {
		return nil, fmt.Errorf("recurrence series is required")
	}

	if err := validateTaskRecurrenceSeries(series); err != nil {
		return nil, err
	}

	if current == nil {
		return nil, fmt.Errorf("current recurrence occurrence is required")
	}

	if current.Sequence < 1 {
		return nil, fmt.Errorf("current occurrence sequence must be at least 1")
	}

	if current.ScheduledDueDate.IsZero() {
		return nil, fmt.Errorf("current scheduled due date is required")
	}

	if reference.IsZero() {
		return nil, fmt.Errorf("recurrence reference time is required")
	}

	if series.Paused {
		return nil, nil
	}

	rule, err := taskRecurrenceRuleFromSeries(series)
	if err != nil {
		return nil, err
	}

	var scheduled time.Time

	switch series.Basis {
	case TaskRecurrenceBasisSchedule:
		switch series.MissedPolicy {
		case TaskRecurrenceMissedNextFuture:
			scheduled, err = nextTaskRecurrenceAfter(
				rule,
				current.ScheduledDueDate,
				reference,
			)

		case TaskRecurrenceMissedEveryOccurrence:
			scheduled, err = nextTaskRecurrenceOccurrence(
				rule,
				current.ScheduledDueDate,
			)

		default:
			return nil, fmt.Errorf(
				"invalid missed-occurrence policy: %d",
				series.MissedPolicy,
			)
		}

	case TaskRecurrenceBasisCompletion:
		anchor := recurrenceCompletionAnchor(
			reference,
			current.ScheduledDueDate,
		)

		scheduled, err = nextTaskRecurrenceOccurrence(
			rule,
			anchor,
		)

	default:
		return nil, fmt.Errorf(
			"invalid recurrence basis: %d",
			series.Basis,
		)
	}

	if err != nil {
		return nil, err
	}

	sequence := current.Sequence + 1

	allowed, err := taskRecurrenceSeriesAllowsOccurrence(
		series,
		sequence,
		scheduled,
	)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, nil
	}

	due, err := taskRecurrenceSeriesDueDate(series, scheduled)
	if err != nil {
		return nil, err
	}

	createAt, err := taskRecurrenceSeriesCreateAt(series, due)
	if err != nil {
		return nil, err
	}

	return &TaskRecurrenceSeriesOccurrencePlan{
		Sequence:         sequence,
		ScheduledDueDate: scheduled,
		DueDate:          due,
		CreateAt:         createAt,
	}, nil
}
