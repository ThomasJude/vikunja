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

func validateYearlyRecurrence(rule *TaskRecurrence) error {
	if rule.Interval < 1 {
		return fmt.Errorf("recurrence interval must be at least 1")
	}

	if rule.Basis != TaskRecurrenceBasisSchedule &&
		rule.Basis != TaskRecurrenceBasisCompletion {
		return fmt.Errorf("invalid recurrence basis: %d", rule.Basis)
	}

	if rule.ByMonth < int(time.January) ||
		rule.ByMonth > int(time.December) {
		return fmt.Errorf("yearly recurrence must define a month")
	}

	hasMonthDay := rule.ByMonthDay != 0
	hasOrdinal := rule.ByWeekdays != 0 || rule.BySetPos != 0

	if hasMonthDay == hasOrdinal {
		return fmt.Errorf(
			"yearly recurrence must define either a month day or an ordinal weekday",
		)
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
			return fmt.Errorf(
				"invalid missing-date policy for yearly month-day recurrence",
			)
		}

		return nil
	}

	if _, ok := recurrenceWeekdayFromMask(rule.ByWeekdays); !ok {
		return fmt.Errorf(
			"ordinal yearly recurrence must define exactly one weekday",
		)
	}

	switch rule.BySetPos {
	case -1, 1, 2, 3, 4, 5:
	default:
		return fmt.Errorf(
			"ordinal position must be 1 through 5 or -1",
		)
	}

	switch rule.MissingPolicy {
	case TaskRecurrenceMissingPolicyDefault,
		TaskRecurrenceMissingPolicySkip,
		TaskRecurrenceMissingPolicyLastOccurrence,
		TaskRecurrenceMissingPolicyNextPeriod:
	default:
		return fmt.Errorf(
			"invalid missing-date policy for yearly ordinal recurrence",
		)
	}

	return nil
}

func yearlyRecurrencePeriod(
	year int,
	month time.Month,
	reference time.Time,
) time.Time {
	return time.Date(
		year,
		month,
		1,
		reference.Hour(),
		reference.Minute(),
		reference.Second(),
		reference.Nanosecond(),
		reference.Location(),
	)
}

func resolveYearlyOccurrence(
	rule *TaskRecurrence,
	target time.Time,
) (time.Time, bool) {
	if rule.ByMonthDay != 0 {
		return resolveMonthDayOccurrence(rule, target)
	}

	return resolveOrdinalWeekdayOccurrence(rule, target)
}

func yearlyRecurrenceNominalYear(
	rule *TaskRecurrence,
	occurrence time.Time,
) int {
	year := occurrence.Year()

	if rule.BySetPos != 5 ||
		rule.MissingPolicy != TaskRecurrenceMissingPolicyNextPeriod {
		return year
	}

	weekday, ok := recurrenceWeekdayFromMask(rule.ByWeekdays)
	if !ok {
		return year
	}

	selectedMonth := time.Month(rule.ByMonth)
	fallbackMonth := selectedMonth + 1
	nominalYear := occurrence.Year()

	if selectedMonth == time.December {
		fallbackMonth = time.January
		nominalYear--
	}

	if occurrence.Month() != fallbackMonth {
		return year
	}

	firstDay, found := ordinalWeekdayInMonth(
		occurrence.Year(),
		occurrence.Month(),
		weekday,
		1,
		occurrence.Location(),
	)
	if !found || occurrence.Day() != firstDay {
		return year
	}

	_, hasFifth := ordinalWeekdayInMonth(
		nominalYear,
		selectedMonth,
		weekday,
		5,
		occurrence.Location(),
	)
	if !hasFifth {
		return nominalYear
	}

	return year
}

func nextYearlyRecurrenceOccurrence(
	rule *TaskRecurrence,
	anchor time.Time,
) (time.Time, error) {
	if anchor.IsZero() {
		return time.Time{}, fmt.Errorf("recurrence anchor is required")
	}

	if err := validateYearlyRecurrence(rule); err != nil {
		return time.Time{}, err
	}

	targetYear := anchor.Year() + rule.Interval

	for range recurrenceSearchLimit {
		target := yearlyRecurrencePeriod(
			targetYear,
			time.Month(rule.ByMonth),
			anchor,
		)

		occurrence, found := resolveYearlyOccurrence(rule, target)
		if found {
			return occurrence, nil
		}

		targetYear += rule.Interval
	}

	return time.Time{}, fmt.Errorf(
		"could not find a valid yearly recurrence occurrence",
	)
}

func nextYearlyRecurrenceAfter(
	rule *TaskRecurrence,
	current time.Time,
	after time.Time,
) (time.Time, error) {
	if current.IsZero() {
		return time.Time{}, fmt.Errorf(
			"current recurrence occurrence is required",
		)
	}

	if err := validateYearlyRecurrence(rule); err != nil {
		return time.Time{}, err
	}

	targetYear := yearlyRecurrenceNominalYear(rule, current) +
		rule.Interval

	for range recurrenceSearchLimit {
		target := yearlyRecurrencePeriod(
			targetYear,
			time.Month(rule.ByMonth),
			current,
		)

		occurrence, found := resolveYearlyOccurrence(rule, target)
		if found && occurrence.After(after) {
			return occurrence, nil
		}

		targetYear += rule.Interval
	}

	return time.Time{}, fmt.Errorf(
		"could not find a future yearly recurrence occurrence",
	)
}
