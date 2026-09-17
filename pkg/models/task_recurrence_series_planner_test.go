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

func TestTaskRecurrenceSeriesPlanner(t *testing.T) {
	location, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	dailySeries := func() *TaskRecurrenceSeries {
		return &TaskRecurrenceSeries{
			RootTaskID:  1,
			ProjectID:   1,
			CreatedByID: 1,

			Frequency: TaskRecurrenceFrequencyDay,
			Interval:  2,
			Basis:     TaskRecurrenceBasisSchedule,

			StartDate: time.Date(
				2026,
				time.September,
				1,
				9,
				0,
				0,
				0,
				location,
			),

			EndType: TaskRecurrenceEndNever,

			WeekendPolicy: TaskRecurrenceWeekendKeep,
			MissedPolicy:  TaskRecurrenceMissedNextFuture,
		}
	}

	currentOccurrence := func() *TaskRecurrenceOccurrence {
		due := time.Date(
			2026,
			time.September,
			1,
			9,
			0,
			0,
			0,
			location,
		)

		return &TaskRecurrenceOccurrence{
			SeriesID: 1,
			TaskID:   1,
			Sequence: 1,

			ScheduledDueDate: due,
			DueDate:          due,
		}
	}

	t.Run("fixed schedule skips historical occurrences by default", func(t *testing.T) {
		series := dailySeries()
		current := currentOccurrence()

		reference := time.Date(
			2026,
			time.September,
			8,
			12,
			0,
			0,
			0,
			location,
		)

		plan, err := planNextTaskRecurrenceSeriesOccurrence(
			series,
			current,
			reference,
		)
		require.NoError(t, err)
		require.NotNil(t, plan)

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

		assert.Equal(t, 2, plan.Sequence)
		assert.True(t, plan.ScheduledDueDate.Equal(expected))
		assert.True(t, plan.DueDate.Equal(expected))
	})

	t.Run("generate every missed occurrence returns next historical occurrence", func(t *testing.T) {
		series := dailySeries()
		series.MissedPolicy = TaskRecurrenceMissedEveryOccurrence

		current := currentOccurrence()

		reference := time.Date(
			2026,
			time.September,
			8,
			12,
			0,
			0,
			0,
			location,
		)

		plan, err := planNextTaskRecurrenceSeriesOccurrence(
			series,
			current,
			reference,
		)
		require.NoError(t, err)
		require.NotNil(t, plan)

		expected := time.Date(
			2026,
			time.September,
			3,
			9,
			0,
			0,
			0,
			location,
		)

		assert.True(t, plan.ScheduledDueDate.Equal(expected))
	})

	t.Run("completion basis preserves scheduled wall clock time", func(t *testing.T) {
		series := dailySeries()
		series.Basis = TaskRecurrenceBasisCompletion

		current := currentOccurrence()

		completedAt := time.Date(
			2026,
			time.September,
			8,
			14,
			30,
			0,
			0,
			location,
		)

		plan, err := planNextTaskRecurrenceSeriesOccurrence(
			series,
			current,
			completedAt,
		)
		require.NoError(t, err)
		require.NotNil(t, plan)

		expected := time.Date(
			2026,
			time.September,
			10,
			9,
			0,
			0,
			0,
			location,
		)

		assert.True(t, plan.ScheduledDueDate.Equal(expected))
	})

	t.Run("weekend handling preserves nominal scheduled date", func(t *testing.T) {
		series := dailySeries()
		series.Interval = 1
		series.WeekendPolicy = TaskRecurrenceWeekendNextBusinessDay
		series.CreateBeforeDays = 3

		friday := time.Date(
			2026,
			time.September,
			18,
			9,
			0,
			0,
			0,
			location,
		)

		current := currentOccurrence()
		current.ScheduledDueDate = friday
		current.DueDate = friday

		plan, err := planNextTaskRecurrenceSeriesOccurrence(
			series,
			current,
			friday,
		)
		require.NoError(t, err)
		require.NotNil(t, plan)

		scheduled := time.Date(
			2026,
			time.September,
			19,
			9,
			0,
			0,
			0,
			location,
		)

		due := time.Date(
			2026,
			time.September,
			21,
			9,
			0,
			0,
			0,
			location,
		)

		createAt := time.Date(
			2026,
			time.September,
			18,
			9,
			0,
			0,
			0,
			location,
		)

		assert.True(t, plan.ScheduledDueDate.Equal(scheduled))
		assert.True(t, plan.DueDate.Equal(due))
		assert.True(t, plan.CreateAt.Equal(createAt))
	})

	t.Run("paused series creates no plan", func(t *testing.T) {
		series := dailySeries()
		series.Paused = true

		plan, err := planNextTaskRecurrenceSeriesOccurrence(
			series,
			currentOccurrence(),
			series.StartDate,
		)
		require.NoError(t, err)
		assert.Nil(t, plan)
	})

	t.Run("end date stops series", func(t *testing.T) {
		series := dailySeries()
		series.EndType = TaskRecurrenceEndDate
		series.EndDate = time.Date(
			2026,
			time.September,
			2,
			9,
			0,
			0,
			0,
			location,
		)

		plan, err := planNextTaskRecurrenceSeriesOccurrence(
			series,
			currentOccurrence(),
			series.StartDate,
		)
		require.NoError(t, err)
		assert.Nil(t, plan)
	})

	t.Run("occurrence count stops series", func(t *testing.T) {
		series := dailySeries()
		series.EndType = TaskRecurrenceEndOccurrences
		series.EndAfterOccurrences = 1

		plan, err := planNextTaskRecurrenceSeriesOccurrence(
			series,
			currentOccurrence(),
			series.StartDate,
		)
		require.NoError(t, err)
		assert.Nil(t, plan)
	})

	t.Run("monthly calculator plugs into series planner", func(t *testing.T) {
		series := dailySeries()
		series.Frequency = TaskRecurrenceFrequencyMonth
		series.Interval = 1
		series.ByMonthDay = 31
		series.MissingPolicy = TaskRecurrenceMissingPolicySkip

		august := time.Date(
			2026,
			time.August,
			31,
			9,
			0,
			0,
			0,
			location,
		)

		current := currentOccurrence()
		current.ScheduledDueDate = august
		current.DueDate = august

		reference := time.Date(
			2026,
			time.September,
			17,
			12,
			0,
			0,
			0,
			location,
		)

		plan, err := planNextTaskRecurrenceSeriesOccurrence(
			series,
			current,
			reference,
		)
		require.NoError(t, err)
		require.NotNil(t, plan)

		expected := time.Date(
			2026,
			time.October,
			31,
			9,
			0,
			0,
			0,
			location,
		)

		assert.True(t, plan.ScheduledDueDate.Equal(expected))
	})

	t.Run("completion basis cannot generate every missed occurrence", func(t *testing.T) {
		series := dailySeries()
		series.Basis = TaskRecurrenceBasisCompletion
		series.MissedPolicy = TaskRecurrenceMissedEveryOccurrence

		err := validateTaskRecurrenceSeries(series)
		assert.EqualError(
			t,
			err,
			"generate every missed occurrence requires repeat on schedule",
		)
	})
}
