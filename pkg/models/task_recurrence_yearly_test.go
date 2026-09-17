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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func yearlyWeekdayMask(day time.Weekday) int {
	return 1 << uint(day)
}

func TestNextYearlyRecurrenceOccurrence(t *testing.T) {
	location, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	t.Run("every year on september seventh", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyYear,
			Interval:   1,
			Basis:      TaskRecurrenceBasisSchedule,
			ByMonth:    int(time.September),
			ByMonthDay: 7,
		}

		current := time.Date(
			2026, time.September, 7, 9, 0, 0, 0, location,
		)

		next, err := nextTaskRecurrenceOccurrence(rule, current)
		require.NoError(t, err)

		expected := time.Date(
			2027, time.September, 7, 9, 0, 0, 0, location,
		)
		assert.True(t, next.Equal(expected))
	})

	t.Run("every two years", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyYear,
			Interval:   2,
			Basis:      TaskRecurrenceBasisSchedule,
			ByMonth:    int(time.September),
			ByMonthDay: 7,
		}

		current := time.Date(
			2026, time.September, 7, 9, 0, 0, 0, location,
		)

		next, err := nextTaskRecurrenceOccurrence(rule, current)
		require.NoError(t, err)

		expected := time.Date(
			2028, time.September, 7, 9, 0, 0, 0, location,
		)
		assert.True(t, next.Equal(expected))
	})

	t.Run("leap day uses last valid day by default", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyYear,
			Interval:   1,
			Basis:      TaskRecurrenceBasisSchedule,
			ByMonth:    int(time.February),
			ByMonthDay: 29,
		}

		current := time.Date(
			2024, time.February, 29, 9, 0, 0, 0, location,
		)

		next, err := nextTaskRecurrenceOccurrence(rule, current)
		require.NoError(t, err)

		expected := time.Date(
			2025, time.February, 28, 9, 0, 0, 0, location,
		)
		assert.True(t, next.Equal(expected))
	})

	t.Run("leap day can skip invalid years", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:     TaskRecurrenceFrequencyYear,
			Interval:      1,
			Basis:         TaskRecurrenceBasisSchedule,
			ByMonth:       int(time.February),
			ByMonthDay:    29,
			MissingPolicy: TaskRecurrenceMissingPolicySkip,
		}

		current := time.Date(
			2024, time.February, 29, 9, 0, 0, 0, location,
		)

		next, err := nextTaskRecurrenceOccurrence(rule, current)
		require.NoError(t, err)

		expected := time.Date(
			2028, time.February, 29, 9, 0, 0, 0, location,
		)
		assert.True(t, next.Equal(expected))
	})

	t.Run("first monday of january", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyYear,
			Interval:   1,
			Basis:      TaskRecurrenceBasisSchedule,
			ByMonth:    int(time.January),
			ByWeekdays: yearlyWeekdayMask(time.Monday),
			BySetPos:   1,
		}

		current := time.Date(
			2026, time.January, 5, 9, 0, 0, 0, location,
		)

		next, err := nextTaskRecurrenceOccurrence(rule, current)
		require.NoError(t, err)

		expected := time.Date(
			2027, time.January, 4, 9, 0, 0, 0, location,
		)
		assert.True(t, next.Equal(expected))
	})

	t.Run("last friday of december", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyYear,
			Interval:   1,
			Basis:      TaskRecurrenceBasisSchedule,
			ByMonth:    int(time.December),
			ByWeekdays: yearlyWeekdayMask(time.Friday),
			BySetPos:   -1,
		}

		current := time.Date(
			2026, time.December, 25, 9, 0, 0, 0, location,
		)

		next, err := nextTaskRecurrenceOccurrence(rule, current)
		require.NoError(t, err)

		expected := time.Date(
			2027, time.December, 31, 9, 0, 0, 0, location,
		)
		assert.True(t, next.Equal(expected))
	})

	t.Run("missing fifth friday uses last occurrence", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:     TaskRecurrenceFrequencyYear,
			Interval:      1,
			Basis:         TaskRecurrenceBasisSchedule,
			ByMonth:       int(time.May),
			ByWeekdays:    yearlyWeekdayMask(time.Friday),
			BySetPos:      5,
			MissingPolicy: TaskRecurrenceMissingPolicyLastOccurrence,
		}

		current := time.Date(
			2026, time.May, 29, 9, 0, 0, 0, location,
		)

		next, err := nextTaskRecurrenceOccurrence(rule, current)
		require.NoError(t, err)

		expected := time.Date(
			2027, time.May, 28, 9, 0, 0, 0, location,
		)
		assert.True(t, next.Equal(expected))
	})

	t.Run("missing fifth friday uses first following month", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:     TaskRecurrenceFrequencyYear,
			Interval:      1,
			Basis:         TaskRecurrenceBasisSchedule,
			ByMonth:       int(time.May),
			ByWeekdays:    yearlyWeekdayMask(time.Friday),
			BySetPos:      5,
			MissingPolicy: TaskRecurrenceMissingPolicyNextPeriod,
		}

		current := time.Date(
			2026, time.May, 29, 9, 0, 0, 0, location,
		)

		next, err := nextTaskRecurrenceOccurrence(rule, current)
		require.NoError(t, err)

		expected := time.Date(
			2027, time.June, 4, 9, 0, 0, 0, location,
		)
		assert.True(t, next.Equal(expected))
	})

	t.Run("december next-period fallback preserves nominal year", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:     TaskRecurrenceFrequencyYear,
			Interval:      1,
			Basis:         TaskRecurrenceBasisSchedule,
			ByMonth:       int(time.December),
			ByWeekdays:    yearlyWeekdayMask(time.Friday),
			BySetPos:      5,
			MissingPolicy: TaskRecurrenceMissingPolicyNextPeriod,
		}

		current := time.Date(
			2027, time.January, 1, 9, 0, 0, 0, location,
		)

		next, err := nextTaskRecurrenceAfter(
			rule,
			current,
			current,
		)
		require.NoError(t, err)

		expected := time.Date(
			2027, time.December, 31, 9, 0, 0, 0, location,
		)
		assert.True(t, next.Equal(expected))
	})
}

func TestNextYearlyRecurrenceAfter(t *testing.T) {
	location, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	rule := &TaskRecurrence{
		Frequency:  TaskRecurrenceFrequencyYear,
		Interval:   2,
		Basis:      TaskRecurrenceBasisSchedule,
		ByMonth:    int(time.September),
		ByMonthDay: 7,
	}

	current := time.Date(
		2024, time.September, 7, 9, 0, 0, 0, location,
	)

	after := time.Date(
		2027, time.October, 1, 9, 0, 0, 0, location,
	)

	next, err := nextTaskRecurrenceAfter(rule, current, after)
	require.NoError(t, err)

	expected := time.Date(
		2028, time.September, 7, 9, 0, 0, 0, location,
	)

	assert.True(t, next.Equal(expected))
}

func TestValidateYearlyRecurrence(t *testing.T) {
	t.Run("specific date valid", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyYear,
			Interval:   1,
			Basis:      TaskRecurrenceBasisSchedule,
			ByMonth:    int(time.September),
			ByMonthDay: 7,
		}

		require.NoError(t, validateTaskRecurrence(rule))
	})

	t.Run("relative date valid", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyYear,
			Interval:   1,
			Basis:      TaskRecurrenceBasisSchedule,
			ByMonth:    int(time.January),
			ByWeekdays: yearlyWeekdayMask(time.Monday),
			BySetPos:   1,
		}

		require.NoError(t, validateTaskRecurrence(rule))
	})

	t.Run("month required", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyYear,
			Interval:   1,
			Basis:      TaskRecurrenceBasisSchedule,
			ByMonthDay: 7,
		}

		require.Error(t, validateTaskRecurrence(rule))
	})

	t.Run("specific and relative date together rejected", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyYear,
			Interval:   1,
			Basis:      TaskRecurrenceBasisSchedule,
			ByMonth:    int(time.May),
			ByMonthDay: 7,
			ByWeekdays: yearlyWeekdayMask(time.Friday),
			BySetPos:   1,
		}

		require.Error(t, validateTaskRecurrence(rule))
	})

	t.Run("multiple weekdays rejected", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency: TaskRecurrenceFrequencyYear,
			Interval:  1,
			Basis:     TaskRecurrenceBasisSchedule,
			ByMonth:   int(time.May),
			ByWeekdays: yearlyWeekdayMask(time.Monday) |
				yearlyWeekdayMask(time.Friday),
			BySetPos: 1,
		}

		require.Error(t, validateTaskRecurrence(rule))
	})
}
