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

func validTaskRecurrenceSeries() *TaskRecurrenceSeries {
	return &TaskRecurrenceSeries{
		RootTaskID:  1,
		ProjectID:   1,
		CreatedByID: 1,

		Frequency: TaskRecurrenceFrequencyMonth,
		Interval:  1,
		Basis:     TaskRecurrenceBasisSchedule,

		StartDate: time.Date(2026, time.September, 17, 9, 0, 0, 0, time.UTC),

		EndType: TaskRecurrenceEndNever,

		WeekendPolicy: TaskRecurrenceWeekendKeep,
		MissedPolicy:  TaskRecurrenceMissedNextFuture,
	}
}

func TestValidateTaskRecurrenceSeries(t *testing.T) {
	t.Run("valid never-ending series", func(t *testing.T) {
		require.NoError(t, validateTaskRecurrenceSeries(validTaskRecurrenceSeries()))
	})

	t.Run("all core frequencies accepted", func(t *testing.T) {
		for _, frequency := range []TaskRecurrenceFrequency{
			TaskRecurrenceFrequencyDay,
			TaskRecurrenceFrequencyWeek,
			TaskRecurrenceFrequencyMonth,
			TaskRecurrenceFrequencyYear,
		} {
			series := validTaskRecurrenceSeries()
			series.Frequency = frequency
			assert.NoError(t, validateTaskRecurrenceSeries(series))
		}
	})

	t.Run("start date required", func(t *testing.T) {
		series := validTaskRecurrenceSeries()
		series.StartDate = time.Time{}

		assert.EqualError(t, validateTaskRecurrenceSeries(series), "recurrence start date is required")
	})

	t.Run("interval must be positive", func(t *testing.T) {
		series := validTaskRecurrenceSeries()
		series.Interval = 0

		assert.EqualError(t, validateTaskRecurrenceSeries(series), "recurrence interval must be at least 1")
	})

	t.Run("end on date", func(t *testing.T) {
		series := validTaskRecurrenceSeries()
		series.EndType = TaskRecurrenceEndDate
		series.EndDate = series.StartDate.AddDate(0, 1, 0)

		require.NoError(t, validateTaskRecurrenceSeries(series))
	})

	t.Run("end date cannot precede start", func(t *testing.T) {
		series := validTaskRecurrenceSeries()
		series.EndType = TaskRecurrenceEndDate
		series.EndDate = series.StartDate.Add(-time.Hour)

		assert.EqualError(t, validateTaskRecurrenceSeries(series), "end date cannot be before start date")
	})

	t.Run("end after occurrences", func(t *testing.T) {
		series := validTaskRecurrenceSeries()
		series.EndType = TaskRecurrenceEndOccurrences
		series.EndAfterOccurrences = 10

		require.NoError(t, validateTaskRecurrenceSeries(series))
	})

	t.Run("occurrence count required", func(t *testing.T) {
		series := validTaskRecurrenceSeries()
		series.EndType = TaskRecurrenceEndOccurrences

		assert.EqualError(t, validateTaskRecurrenceSeries(series), "occurrence limit must be at least 1")
	})

	t.Run("never cannot contain end settings", func(t *testing.T) {
		series := validTaskRecurrenceSeries()
		series.EndDate = series.StartDate.AddDate(0, 1, 0)

		assert.EqualError(
			t,
			validateTaskRecurrenceSeries(series),
			"never-ending recurrence cannot define an end date or occurrence limit",
		)
	})

	t.Run("create before cannot be negative", func(t *testing.T) {
		series := validTaskRecurrenceSeries()
		series.CreateBeforeDays = -1

		assert.EqualError(t, validateTaskRecurrenceSeries(series), "create-before days cannot be negative")
	})

	t.Run("weekend policies", func(t *testing.T) {
		for _, policy := range []TaskRecurrenceWeekendPolicy{
			TaskRecurrenceWeekendKeep,
			TaskRecurrenceWeekendPreviousBusinessDay,
			TaskRecurrenceWeekendNextBusinessDay,
		} {
			series := validTaskRecurrenceSeries()
			series.WeekendPolicy = policy
			assert.NoError(t, validateTaskRecurrenceSeries(series))
		}
	})

	t.Run("missed occurrence policies", func(t *testing.T) {
		for _, policy := range []TaskRecurrenceMissedPolicy{
			TaskRecurrenceMissedNextFuture,
			TaskRecurrenceMissedEveryOccurrence,
		} {
			series := validTaskRecurrenceSeries()
			series.MissedPolicy = policy
			assert.NoError(t, validateTaskRecurrenceSeries(series))
		}
	})
}
