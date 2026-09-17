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

func validateDailyRecurrence(rule *TaskRecurrence) error {
	if rule.Interval < 1 {
		return fmt.Errorf("recurrence interval must be at least 1")
	}

	if rule.Basis != TaskRecurrenceBasisSchedule &&
		rule.Basis != TaskRecurrenceBasisCompletion {
		return fmt.Errorf("invalid recurrence basis: %d", rule.Basis)
	}

	if rule.ByWeekdays != 0 ||
		rule.ByMonth != 0 ||
		rule.ByMonthDay != 0 ||
		rule.BySetPos != 0 {
		return fmt.Errorf("daily recurrence cannot define calendar selectors")
	}

	if rule.MissingPolicy != TaskRecurrenceMissingPolicyDefault {
		return fmt.Errorf("daily recurrence cannot define a missing-date policy")
	}

	return nil
}

func nextDailyRecurrenceOccurrence(
	rule *TaskRecurrence,
	anchor time.Time,
) (time.Time, error) {
	if anchor.IsZero() {
		return time.Time{}, fmt.Errorf("recurrence anchor is required")
	}

	if err := validateDailyRecurrence(rule); err != nil {
		return time.Time{}, err
	}

	return anchor.AddDate(0, 0, rule.Interval), nil
}

func nextDailyRecurrenceAfter(
	rule *TaskRecurrence,
	current time.Time,
	after time.Time,
) (time.Time, error) {
	if current.IsZero() {
		return time.Time{}, fmt.Errorf("current recurrence occurrence is required")
	}

	if err := validateDailyRecurrence(rule); err != nil {
		return time.Time{}, err
	}

	next := current.AddDate(0, 0, rule.Interval)

	for range recurrenceSearchLimit {
		if next.After(after) {
			return next, nil
		}

		next = next.AddDate(0, 0, rule.Interval)
	}

	return time.Time{}, fmt.Errorf("could not find a future recurrence occurrence")
}
