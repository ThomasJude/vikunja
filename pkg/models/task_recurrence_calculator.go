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

const recurrenceSearchLimit = 4800

func validateTaskRecurrence(rule *TaskRecurrence) error {
	if rule == nil {
		return nil
	}

	switch rule.Frequency {
	case TaskRecurrenceFrequencyMonth:
		if err := validateMonthlyRecurrence(rule); err != nil {
			return ErrInvalidTaskRecurrence{Reason: err.Error()}
		}
	default:
		return ErrInvalidTaskRecurrence{
			Reason: fmt.Sprintf("unsupported recurrence frequency: %d", rule.Frequency),
		}
	}

	return nil
}

func nextTaskRecurrenceOccurrence(rule *TaskRecurrence, anchor time.Time) (time.Time, error) {
	if rule == nil {
		return time.Time{}, fmt.Errorf("recurrence rule is required")
	}

	switch rule.Frequency {
	case TaskRecurrenceFrequencyMonth:
		return nextMonthlyRecurrenceOccurrence(rule, anchor)
	default:
		return time.Time{}, fmt.Errorf("unsupported recurrence frequency: %d", rule.Frequency)
	}
}

func validateMonthlyRecurrence(rule *TaskRecurrence) error {
	if rule.Interval < 1 {
		return fmt.Errorf("recurrence interval must be at least 1")
	}

	if rule.Basis != TaskRecurrenceBasisSchedule &&
		rule.Basis != TaskRecurrenceBasisCompletion {
		return fmt.Errorf("invalid recurrence basis: %d", rule.Basis)
	}

	if rule.ByMonth != 0 {
		return fmt.Errorf("by_month is not valid for monthly recurrence")
	}

	hasMonthDay := rule.ByMonthDay != 0
	hasOrdinal := rule.ByWeekdays != 0 || rule.BySetPos != 0

	if hasMonthDay == hasOrdinal {
		return fmt.Errorf("monthly recurrence must define either a month day or an ordinal weekday")
	}

	if hasMonthDay {
		if rule.ByMonthDay < 1 || rule.ByMonthDay > 31 {
			return fmt.Errorf("month day must be between 1 and 31")
		}

		switch rule.MissingPolicy {
		case TaskRecurrenceMissingPolicyDefault,
			TaskRecurrenceMissingPolicyLastValid,
			TaskRecurrenceMissingPolicySkip:
		default:
			return fmt.Errorf("invalid missing-date policy for month-day recurrence")
		}

		return nil
	}

	if _, ok := recurrenceWeekdayFromMask(rule.ByWeekdays); !ok {
		return fmt.Errorf("ordinal monthly recurrence must define exactly one weekday")
	}

	switch rule.BySetPos {
	case -1, 1, 2, 3, 4, 5:
	default:
		return fmt.Errorf("ordinal position must be 1 through 5 or -1")
	}

	switch rule.MissingPolicy {
	case TaskRecurrenceMissingPolicyDefault,
		TaskRecurrenceMissingPolicySkip,
		TaskRecurrenceMissingPolicyLastOccurrence,
		TaskRecurrenceMissingPolicyNextPeriod:
	default:
		return fmt.Errorf("invalid missing-date policy for ordinal recurrence")
	}

	return nil
}

func nextMonthlyRecurrenceOccurrence(rule *TaskRecurrence, anchor time.Time) (time.Time, error) {
	if anchor.IsZero() {
		return time.Time{}, fmt.Errorf("recurrence anchor is required")
	}

	if err := validateMonthlyRecurrence(rule); err != nil {
		return time.Time{}, err
	}

	target := recurrenceMonthStart(anchor, rule.Interval)

	for range recurrenceSearchLimit {
		occurrence, found := resolveMonthlyOccurrence(rule, target)
		if found {
			return occurrence, nil
		}

		target = recurrenceMonthStart(target, rule.Interval)
	}

	return time.Time{}, fmt.Errorf("could not find a valid recurrence occurrence")
}

func resolveMonthlyOccurrence(rule *TaskRecurrence, target time.Time) (time.Time, bool) {
	if rule.ByMonthDay != 0 {
		return resolveMonthDayOccurrence(rule, target)
	}

	return resolveOrdinalWeekdayOccurrence(rule, target)
}

func resolveMonthDayOccurrence(rule *TaskRecurrence, target time.Time) (time.Time, bool) {
	lastDay := daysInMonth(target.Year(), target.Month(), target.Location())

	if rule.ByMonthDay <= lastDay {
		return recurrenceDate(target, rule.ByMonthDay), true
	}

	switch rule.MissingPolicy {
	case TaskRecurrenceMissingPolicyDefault, TaskRecurrenceMissingPolicyLastValid:
		return recurrenceDate(target, lastDay), true
	case TaskRecurrenceMissingPolicySkip:
		return time.Time{}, false
	}

	return time.Time{}, false
}

func resolveOrdinalWeekdayOccurrence(rule *TaskRecurrence, target time.Time) (time.Time, bool) {
	weekday, ok := recurrenceWeekdayFromMask(rule.ByWeekdays)
	if !ok {
		return time.Time{}, false
	}

	day, found := ordinalWeekdayInMonth(
		target.Year(),
		target.Month(),
		weekday,
		rule.BySetPos,
		target.Location(),
	)

	if found {
		return recurrenceDate(target, day), true
	}

	switch rule.MissingPolicy {
	case TaskRecurrenceMissingPolicyDefault, TaskRecurrenceMissingPolicySkip:
		return time.Time{}, false

	case TaskRecurrenceMissingPolicyLastOccurrence:
		day, _ = ordinalWeekdayInMonth(
			target.Year(),
			target.Month(),
			weekday,
			-1,
			target.Location(),
		)
		return recurrenceDate(target, day), true

	case TaskRecurrenceMissingPolicyNextPeriod:
		nextMonth := recurrenceMonthStart(target, 1)
		day, _ = ordinalWeekdayInMonth(
			nextMonth.Year(),
			nextMonth.Month(),
			weekday,
			1,
			nextMonth.Location(),
		)
		return recurrenceDate(nextMonth, day), true
	}

	return time.Time{}, false
}

func recurrenceMonthStart(t time.Time, months int) time.Time {
	return time.Date(
		t.Year(),
		t.Month()+time.Month(months),
		1,
		t.Hour(),
		t.Minute(),
		t.Second(),
		t.Nanosecond(),
		t.Location(),
	)
}

func recurrenceDate(base time.Time, day int) time.Time {
	return time.Date(
		base.Year(),
		base.Month(),
		day,
		base.Hour(),
		base.Minute(),
		base.Second(),
		base.Nanosecond(),
		base.Location(),
	)
}

func daysInMonth(year int, month time.Month, location *time.Location) int {
	return time.Date(year, month+1, 0, 12, 0, 0, 0, location).Day()
}

func recurrenceWeekdayFromMask(mask int) (time.Weekday, bool) {
	if mask <= 0 || mask > 0b1111111 || mask&(mask-1) != 0 {
		return 0, false
	}

	for day := time.Sunday; day <= time.Saturday; day++ {
		if mask == 1<<uint(day) {
			return day, true
		}
	}

	return 0, false
}

func ordinalWeekdayInMonth(
	year int,
	month time.Month,
	weekday time.Weekday,
	position int,
	location *time.Location,
) (int, bool) {
	lastDay := daysInMonth(year, month, location)

	if position == -1 {
		lastWeekday := time.Date(year, month, lastDay, 12, 0, 0, 0, location).Weekday()
		offset := (int(lastWeekday) - int(weekday) + 7) % 7

		return lastDay - offset, true
	}

	firstWeekday := time.Date(year, month, 1, 12, 0, 0, 0, location).Weekday()
	offset := (int(weekday) - int(firstWeekday) + 7) % 7
	day := 1 + offset + ((position - 1) * 7)

	if day > lastDay {
		return 0, false
	}

	return day, true
}
