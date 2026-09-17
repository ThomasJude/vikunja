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

func TestNextDailyRecurrenceOccurrence(t *testing.T) {
	location := time.FixedZone("PKT", 5*60*60)

	tests := []struct {
		name     string
		interval int
		anchor   time.Time
		expected time.Time
	}{
		{
			name:     "every day",
			interval: 1,
			anchor: time.Date(
				2026,
				time.September,
				17,
				9,
				0,
				0,
				0,
				location,
			),
			expected: time.Date(
				2026,
				time.September,
				18,
				9,
				0,
				0,
				0,
				location,
			),
		},
		{
			name:     "every two days",
			interval: 2,
			anchor: time.Date(
				2026,
				time.September,
				17,
				9,
				0,
				0,
				0,
				location,
			),
			expected: time.Date(
				2026,
				time.September,
				19,
				9,
				0,
				0,
				0,
				location,
			),
		},
		{
			name:     "every thirty days",
			interval: 30,
			anchor: time.Date(
				2026,
				time.September,
				17,
				9,
				0,
				0,
				0,
				location,
			),
			expected: time.Date(
				2026,
				time.October,
				17,
				9,
				0,
				0,
				0,
				location,
			),
		},
		{
			name:     "crosses year boundary",
			interval: 2,
			anchor: time.Date(
				2026,
				time.December,
				31,
				9,
				0,
				0,
				0,
				location,
			),
			expected: time.Date(
				2027,
				time.January,
				2,
				9,
				0,
				0,
				0,
				location,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := &TaskRecurrence{
				Frequency: TaskRecurrenceFrequencyDay,
				Interval:  tt.interval,
				Basis:     TaskRecurrenceBasisSchedule,
			}

			next, err := nextTaskRecurrenceOccurrence(rule, tt.anchor)
			require.NoError(t, err)
			assert.True(
				t,
				next.Equal(tt.expected),
				"expected %s, got %s",
				tt.expected,
				next,
			)
		})
	}
}

func TestNextDailyRecurrenceAfter(t *testing.T) {
	location := time.FixedZone("PKT", 5*60*60)

	t.Run("schedule skips historical occurrences", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency: TaskRecurrenceFrequencyDay,
			Interval:  2,
			Basis:     TaskRecurrenceBasisSchedule,
		}

		current := time.Date(
			2026,
			time.September,
			1,
			9,
			0,
			0,
			0,
			location,
		)

		after := time.Date(
			2026,
			time.September,
			8,
			12,
			0,
			0,
			0,
			location,
		)

		next, err := nextTaskRecurrenceAfter(
			rule,
			current,
			after,
		)
		require.NoError(t, err)

		expected := time.Date(
			2026,
			time.September,
			9,
			9,
			0,
			0,
			0,
			location,
		)

		assert.True(
			t,
			next.Equal(expected),
			"expected %s, got %s",
			expected,
			next,
		)
	})
}

func TestValidateDailyRecurrence(t *testing.T) {
	t.Run("valid daily rule", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency: TaskRecurrenceFrequencyDay,
			Interval:  30,
			Basis:     TaskRecurrenceBasisCompletion,
		}

		require.NoError(t, validateTaskRecurrence(rule))
	})

	t.Run("zero interval", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency: TaskRecurrenceFrequencyDay,
			Interval:  0,
			Basis:     TaskRecurrenceBasisSchedule,
		}

		require.Error(t, validateTaskRecurrence(rule))
	})

	t.Run("calendar selectors rejected", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyDay,
			Interval:   1,
			Basis:      TaskRecurrenceBasisSchedule,
			ByWeekdays: 1,
		}

		require.Error(t, validateTaskRecurrence(rule))
	})

	t.Run("missing policy rejected", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:     TaskRecurrenceFrequencyDay,
			Interval:      1,
			Basis:         TaskRecurrenceBasisSchedule,
			MissingPolicy: TaskRecurrenceMissingPolicySkip,
		}

		require.Error(t, validateTaskRecurrence(rule))
	})
}
