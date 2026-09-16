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

func TestNextMonthlyRecurrenceOccurrence(t *testing.T) {
	t.Run("day of month", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyMonth,
			Interval:   1,
			ByMonthDay: 15,
		}

		next, err := nextTaskRecurrenceOccurrence(
			rule,
			time.Date(2026, time.January, 15, 9, 30, 0, 0, time.UTC),
		)

		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, time.February, 15, 9, 30, 0, 0, time.UTC), next)
	})

	t.Run("quarterly", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyMonth,
			Interval:   3,
			ByMonthDay: 15,
		}

		next, err := nextTaskRecurrenceOccurrence(
			rule,
			time.Date(2026, time.January, 15, 14, 45, 0, 0, time.UTC),
		)

		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, time.April, 15, 14, 45, 0, 0, time.UTC), next)
	})

	t.Run("31st uses last day by default", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyMonth,
			Interval:   1,
			ByMonthDay: 31,
		}

		next, err := nextTaskRecurrenceOccurrence(
			rule,
			time.Date(2025, time.January, 31, 10, 0, 0, 0, time.UTC),
		)

		require.NoError(t, err)
		assert.Equal(t, time.Date(2025, time.February, 28, 10, 0, 0, 0, time.UTC), next)
	})

	t.Run("31st uses leap day", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyMonth,
			Interval:   1,
			ByMonthDay: 31,
		}

		next, err := nextTaskRecurrenceOccurrence(
			rule,
			time.Date(2024, time.January, 31, 10, 0, 0, 0, time.UTC),
		)

		require.NoError(t, err)
		assert.Equal(t, time.Date(2024, time.February, 29, 10, 0, 0, 0, time.UTC), next)
	})

	t.Run("31st skips invalid month", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:     TaskRecurrenceFrequencyMonth,
			Interval:      1,
			ByMonthDay:    31,
			MissingPolicy: TaskRecurrenceMissingPolicySkip,
		}

		next, err := nextTaskRecurrenceOccurrence(
			rule,
			time.Date(2025, time.January, 31, 10, 0, 0, 0, time.UTC),
		)

		require.NoError(t, err)
		assert.Equal(t, time.Date(2025, time.March, 31, 10, 0, 0, 0, time.UTC), next)
	})

	t.Run("year boundary", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyMonth,
			Interval:   2,
			ByMonthDay: 31,
		}

		next, err := nextTaskRecurrenceOccurrence(
			rule,
			time.Date(2026, time.November, 30, 8, 0, 0, 0, time.UTC),
		)

		require.NoError(t, err)
		assert.Equal(t, time.Date(2027, time.January, 31, 8, 0, 0, 0, time.UTC), next)
	})

	t.Run("first monday", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyMonth,
			Interval:   1,
			ByWeekdays: 1 << uint(time.Monday),
			BySetPos:   1,
		}

		next, err := nextTaskRecurrenceOccurrence(
			rule,
			time.Date(2026, time.January, 10, 9, 0, 0, 0, time.UTC),
		)

		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, time.February, 2, 9, 0, 0, 0, time.UTC), next)
	})

	t.Run("third thursday", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyMonth,
			Interval:   1,
			ByWeekdays: 1 << uint(time.Thursday),
			BySetPos:   3,
		}

		next, err := nextTaskRecurrenceOccurrence(
			rule,
			time.Date(2026, time.January, 10, 9, 0, 0, 0, time.UTC),
		)

		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, time.February, 19, 9, 0, 0, 0, time.UTC), next)
	})

	t.Run("last friday", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyMonth,
			Interval:   1,
			ByWeekdays: 1 << uint(time.Friday),
			BySetPos:   -1,
		}

		next, err := nextTaskRecurrenceOccurrence(
			rule,
			time.Date(2026, time.January, 10, 9, 0, 0, 0, time.UTC),
		)

		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, time.February, 27, 9, 0, 0, 0, time.UTC), next)
	})

	t.Run("missing fifth weekday skips by default", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyMonth,
			Interval:   1,
			ByWeekdays: 1 << uint(time.Monday),
			BySetPos:   5,
		}

		next, err := nextTaskRecurrenceOccurrence(
			rule,
			time.Date(2026, time.January, 10, 9, 0, 0, 0, time.UTC),
		)

		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, time.March, 30, 9, 0, 0, 0, time.UTC), next)
	})

	t.Run("missing fifth weekday uses last occurrence", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:     TaskRecurrenceFrequencyMonth,
			Interval:      1,
			ByWeekdays:    1 << uint(time.Monday),
			BySetPos:      5,
			MissingPolicy: TaskRecurrenceMissingPolicyLastOccurrence,
		}

		next, err := nextTaskRecurrenceOccurrence(
			rule,
			time.Date(2026, time.January, 10, 9, 0, 0, 0, time.UTC),
		)

		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, time.February, 23, 9, 0, 0, 0, time.UTC), next)
	})

	t.Run("missing fifth weekday uses first occurrence next month", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:     TaskRecurrenceFrequencyMonth,
			Interval:      1,
			ByWeekdays:    1 << uint(time.Monday),
			BySetPos:      5,
			MissingPolicy: TaskRecurrenceMissingPolicyNextPeriod,
		}

		next, err := nextTaskRecurrenceOccurrence(
			rule,
			time.Date(2026, time.January, 10, 9, 0, 0, 0, time.UTC),
		)

		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, time.March, 2, 9, 0, 0, 0, time.UTC), next)
	})

	t.Run("valid fifth monday", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyMonth,
			Interval:   1,
			ByWeekdays: 1 << uint(time.Monday),
			BySetPos:   5,
		}

		next, err := nextTaskRecurrenceOccurrence(
			rule,
			time.Date(2026, time.May, 10, 9, 0, 0, 0, time.UTC),
		)

		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, time.June, 29, 9, 0, 0, 0, time.UTC), next)
	})

	t.Run("skip preserves multi month cadence", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:     TaskRecurrenceFrequencyMonth,
			Interval:      2,
			ByMonthDay:    31,
			MissingPolicy: TaskRecurrenceMissingPolicySkip,
		}

		next, err := nextTaskRecurrenceOccurrence(
			rule,
			time.Date(2025, time.December, 31, 11, 15, 0, 0, time.UTC),
		)

		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, time.August, 31, 11, 15, 0, 0, time.UTC), next)
	})

	t.Run("explicit last valid day", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:     TaskRecurrenceFrequencyMonth,
			Interval:      1,
			ByMonthDay:    30,
			MissingPolicy: TaskRecurrenceMissingPolicyLastValid,
		}

		next, err := nextTaskRecurrenceOccurrence(
			rule,
			time.Date(2024, time.January, 30, 16, 20, 0, 0, time.UTC),
		)

		require.NoError(t, err)
		assert.Equal(t, time.Date(2024, time.February, 29, 16, 20, 0, 0, time.UTC), next)
	})
}

func TestValidateMonthlyRecurrence(t *testing.T) {
	tests := []struct {
		name string
		rule TaskRecurrence
	}{
		{
			name: "zero interval",
			rule: TaskRecurrence{
				Frequency:  TaskRecurrenceFrequencyMonth,
				ByMonthDay: 1,
			},
		},
		{
			name: "invalid month day",
			rule: TaskRecurrence{
				Frequency:  TaskRecurrenceFrequencyMonth,
				Interval:   1,
				ByMonthDay: 32,
			},
		},
		{
			name: "month day and ordinal together",
			rule: TaskRecurrence{
				Frequency:  TaskRecurrenceFrequencyMonth,
				Interval:   1,
				ByMonthDay: 15,
				ByWeekdays: 1 << uint(time.Monday),
				BySetPos:   1,
			},
		},
		{
			name: "multiple ordinal weekdays",
			rule: TaskRecurrence{
				Frequency:  TaskRecurrenceFrequencyMonth,
				Interval:   1,
				ByWeekdays: 1<<uint(time.Monday) | 1<<uint(time.Friday),
				BySetPos:   1,
			},
		},
		{
			name: "invalid ordinal",
			rule: TaskRecurrence{
				Frequency:  TaskRecurrenceFrequencyMonth,
				Interval:   1,
				ByWeekdays: 1 << uint(time.Monday),
				BySetPos:   6,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, validateMonthlyRecurrence(&tt.rule))
		})
	}
}
