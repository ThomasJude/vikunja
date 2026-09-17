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

func recurrenceWeekdayMask(days ...time.Weekday) int {
	mask := 0
	for _, day := range days {
		mask |= 1 << uint(day)
	}
	return mask
}

func TestNextWeeklyRecurrenceOccurrence(t *testing.T) {
	location, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	t.Run("every monday", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyWeek,
			Interval:   1,
			Basis:      TaskRecurrenceBasisSchedule,
			ByWeekdays: recurrenceWeekdayMask(time.Monday),
		}

		current := time.Date(
			2026, time.September, 14, 9, 0, 0, 0, location,
		)

		next, err := nextTaskRecurrenceOccurrence(rule, current)
		require.NoError(t, err)

		expected := time.Date(
			2026, time.September, 21, 9, 0, 0, 0, location,
		)

		assert.True(t, next.Equal(expected))
	})

	t.Run("monday wednesday friday", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency: TaskRecurrenceFrequencyWeek,
			Interval:  1,
			Basis:     TaskRecurrenceBasisSchedule,
			ByWeekdays: recurrenceWeekdayMask(
				time.Monday,
				time.Wednesday,
				time.Friday,
			),
		}

		monday := time.Date(
			2026, time.September, 14, 9, 0, 0, 0, location,
		)

		wednesday, err := nextTaskRecurrenceOccurrence(rule, monday)
		require.NoError(t, err)

		assert.True(
			t,
			wednesday.Equal(
				time.Date(
					2026,
					time.September,
					16,
					9,
					0,
					0,
					0,
					location,
				),
			),
		)

		friday, err := nextTaskRecurrenceOccurrence(rule, wednesday)
		require.NoError(t, err)

		assert.True(
			t,
			friday.Equal(
				time.Date(
					2026,
					time.September,
					18,
					9,
					0,
					0,
					0,
					location,
				),
			),
		)

		nextMonday, err := nextTaskRecurrenceOccurrence(rule, friday)
		require.NoError(t, err)

		assert.True(
			t,
			nextMonday.Equal(
				time.Date(
					2026,
					time.September,
					21,
					9,
					0,
					0,
					0,
					location,
				),
			),
		)
	})

	t.Run("every two weeks on tuesday", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyWeek,
			Interval:   2,
			Basis:      TaskRecurrenceBasisSchedule,
			ByWeekdays: recurrenceWeekdayMask(time.Tuesday),
		}

		current := time.Date(
			2026, time.September, 15, 9, 0, 0, 0, location,
		)

		next, err := nextTaskRecurrenceOccurrence(rule, current)
		require.NoError(t, err)

		expected := time.Date(
			2026, time.September, 29, 9, 0, 0, 0, location,
		)

		assert.True(t, next.Equal(expected))
	})

	t.Run("every three weeks monday and thursday", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency: TaskRecurrenceFrequencyWeek,
			Interval:  3,
			Basis:     TaskRecurrenceBasisSchedule,
			ByWeekdays: recurrenceWeekdayMask(
				time.Monday,
				time.Thursday,
			),
		}

		current := time.Date(
			2026, time.September, 17, 9, 0, 0, 0, location,
		)

		next, err := nextTaskRecurrenceOccurrence(rule, current)
		require.NoError(t, err)

		expected := time.Date(
			2026, time.October, 5, 9, 0, 0, 0, location,
		)

		assert.True(t, next.Equal(expected))
	})

	t.Run("sunday works", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyWeek,
			Interval:   1,
			Basis:      TaskRecurrenceBasisSchedule,
			ByWeekdays: recurrenceWeekdayMask(time.Sunday),
		}

		current := time.Date(
			2026, time.September, 17, 9, 0, 0, 0, location,
		)

		next, err := nextTaskRecurrenceOccurrence(rule, current)
		require.NoError(t, err)

		expected := time.Date(
			2026, time.September, 20, 9, 0, 0, 0, location,
		)

		assert.True(t, next.Equal(expected))
	})
}

func TestNextWeeklyRecurrenceAfter(t *testing.T) {
	location, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	rule := &TaskRecurrence{
		Frequency: TaskRecurrenceFrequencyWeek,
		Interval:  1,
		Basis:     TaskRecurrenceBasisSchedule,
		ByWeekdays: recurrenceWeekdayMask(
			time.Monday,
			time.Wednesday,
			time.Friday,
		),
	}

	current := time.Date(
		2026, time.September, 7, 9, 0, 0, 0, location,
	)

	after := time.Date(
		2026, time.September, 17, 12, 0, 0, 0, location,
	)

	next, err := nextTaskRecurrenceAfter(rule, current, after)
	require.NoError(t, err)

	expected := time.Date(
		2026, time.September, 18, 9, 0, 0, 0, location,
	)

	assert.True(t, next.Equal(expected))
}

func TestValidateWeeklyRecurrence(t *testing.T) {
	t.Run("multiple weekdays valid", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency: TaskRecurrenceFrequencyWeek,
			Interval:  2,
			Basis:     TaskRecurrenceBasisSchedule,
			ByWeekdays: recurrenceWeekdayMask(
				time.Monday,
				time.Thursday,
			),
		}

		require.NoError(t, validateTaskRecurrence(rule))
	})

	t.Run("weekdays preset mask valid", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency: TaskRecurrenceFrequencyWeek,
			Interval:  1,
			Basis:     TaskRecurrenceBasisSchedule,
			ByWeekdays: recurrenceWeekdayMask(
				time.Monday,
				time.Tuesday,
				time.Wednesday,
				time.Thursday,
				time.Friday,
			),
		}

		require.NoError(t, validateTaskRecurrence(rule))
	})

	t.Run("weekday required", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency: TaskRecurrenceFrequencyWeek,
			Interval:  1,
			Basis:     TaskRecurrenceBasisSchedule,
		}

		require.Error(t, validateTaskRecurrence(rule))
	})

	t.Run("month selectors rejected", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyWeek,
			Interval:   1,
			Basis:      TaskRecurrenceBasisSchedule,
			ByWeekdays: recurrenceWeekdayMask(time.Monday),
			ByMonthDay: 15,
		}

		require.Error(t, validateTaskRecurrence(rule))
	})

	t.Run("missing policy rejected", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:     TaskRecurrenceFrequencyWeek,
			Interval:      1,
			Basis:         TaskRecurrenceBasisSchedule,
			ByWeekdays:    recurrenceWeekdayMask(time.Monday),
			MissingPolicy: TaskRecurrenceMissingPolicySkip,
		}

		require.Error(t, validateTaskRecurrence(rule))
	})
}
