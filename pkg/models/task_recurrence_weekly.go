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

func validateWeeklyRecurrence(rule *TaskRecurrence) error {
	if rule.Interval < 1 {
		return fmt.Errorf("recurrence interval must be at least 1")
	}

	if rule.Basis != TaskRecurrenceBasisSchedule &&
		rule.Basis != TaskRecurrenceBasisCompletion {
		return fmt.Errorf("invalid recurrence basis: %d", rule.Basis)
	}

	if rule.ByWeekdays <= 0 || rule.ByWeekdays > 0b1111111 {
		return fmt.Errorf("weekly recurrence must define at least one weekday")
	}

	if rule.ByMonth != 0 ||
		rule.ByMonthDay != 0 ||
		rule.BySetPos != 0 {
		return fmt.Errorf("weekly recurrence cannot define month selectors")
	}

	if rule.MissingPolicy != TaskRecurrenceMissingPolicyDefault {
		return fmt.Errorf("weekly recurrence cannot define a missing-date policy")
	}

	return nil
}

func recurrenceWeekStartMonday(t time.Time) time.Time {
	offset := (int(t.Weekday()) + 6) % 7

	return t.AddDate(0, 0, -offset)
}

func recurrenceWeekdayOffsetFromMonday(day time.Weekday) int {
	return (int(day) + 6) % 7
}

func weeklyOccurrenceInWeekAfter(
	rule *TaskRecurrence,
	weekStart time.Time,
	after time.Time,
) (time.Time, bool) {
	for day := time.Monday; ; day++ {
		if rule.ByWeekdays&(1<<uint(day)) != 0 {
			candidate := weekStart.AddDate(
				0,
				0,
				recurrenceWeekdayOffsetFromMonday(day),
			)

			if candidate.After(after) {
				return candidate, true
			}
		}

		if day == time.Saturday {
			break
		}
	}

	if rule.ByWeekdays&(1<<uint(time.Sunday)) != 0 {
		candidate := weekStart.AddDate(0, 0, 6)
		if candidate.After(after) {
			return candidate, true
		}
	}

	return time.Time{}, false
}

func firstWeeklyOccurrenceInWeek(
	rule *TaskRecurrence,
	weekStart time.Time,
) (time.Time, bool) {
	beforeWeek := weekStart.Add(-time.Nanosecond)
	return weeklyOccurrenceInWeekAfter(rule, weekStart, beforeWeek)
}

func nextWeeklyRecurrenceOccurrence(
	rule *TaskRecurrence,
	anchor time.Time,
) (time.Time, error) {
	if anchor.IsZero() {
		return time.Time{}, fmt.Errorf("recurrence anchor is required")
	}

	if err := validateWeeklyRecurrence(rule); err != nil {
		return time.Time{}, err
	}

	weekStart := recurrenceWeekStartMonday(anchor)

	if next, found := weeklyOccurrenceInWeekAfter(
		rule,
		weekStart,
		anchor,
	); found {
		return next, nil
	}

	targetWeek := weekStart.AddDate(
		0,
		0,
		rule.Interval*7,
	)

	next, found := firstWeeklyOccurrenceInWeek(rule, targetWeek)
	if !found {
		return time.Time{}, fmt.Errorf("could not find a weekly recurrence occurrence")
	}

	return next, nil
}

func nextWeeklyRecurrenceAfter(
	rule *TaskRecurrence,
	current time.Time,
	after time.Time,
) (time.Time, error) {
	if current.IsZero() {
		return time.Time{}, fmt.Errorf("current recurrence occurrence is required")
	}

	if err := validateWeeklyRecurrence(rule); err != nil {
		return time.Time{}, err
	}

	next := current

	for range recurrenceSearchLimit {
		var err error
		next, err = nextWeeklyRecurrenceOccurrence(rule, next)
		if err != nil {
			return time.Time{}, err
		}

		if next.After(after) {
			return next, nil
		}
	}

	return time.Time{}, fmt.Errorf("could not find a future recurrence occurrence")
}
